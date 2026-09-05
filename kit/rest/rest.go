// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	kr "github.com/go-kratos/kratos/v2/errors"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
)

type RouteHandler func(e *echo.Echo)

// New 创建 REST 服务，调用方负责启动和关闭
func New(app string, routes RouteHandler) (e *echo.Echo) {
	e = echo.New()

	// 隐藏banner
	e.HideBanner = true

	// 隐藏打印端口
	e.HidePort = true

	// 获取真实IP
	e.IPExtractor = echo.ExtractIPFromXFFHeader()

	// 默认json序列化工具
	e.JSONSerializer = NewDefaultJSONSerializer()

	// 绑定校验器
	e.Validator = NewValidator()

	// 默认错误处理
	e.HTTPErrorHandler = handleHTTPError

	// 设置全局中间件
	e.Use(
		ContextMiddleware(app),
		RecoverMiddleware(),
	)

	// 设置路由
	if routes != nil {
		routes(e)
	}

	return
}

// Run 启动 REST 服务，服务停止后关闭错误通道
func Run(app, address string, routes RouteHandler) (e *echo.Echo, ch chan error) {
	e = New(app, routes)

	// 使用协程启动HTTP Rest服务器
	ch = make(chan error, 1)

	go func() {
		defer close(ch)

		if err := e.Start(address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- fmt.Errorf("HTTP REST 服务启动失败：%w", err)
		}
	}()

	return
}

func handleHTTPError(err error, c echo.Context) {
	if err == nil || c.Response().Committed {
		return
	}

	response := NewResponse().SetCode(http.StatusInternalServerError)
	var responseErr *Error
	var httpErr *echo.HTTPError

	switch {
	case errors.As(err, &responseErr):
		response.SetCode(responseErr.Code).SetMessage(responseErr.Message)
	case errors.As(err, &httpErr):
		response.SetCode(httpErr.Code).SetMessage(fmt.Sprint(httpErr.Message))
	default:
		if _, ok := status.FromError(err); ok {
			remote := kr.FromError(err)
			response.SetCode(int(remote.Code)).SetMessage(remote.Message)
		}
	}

	if response.Code < http.StatusBadRequest || response.Code > 599 {
		response.SetCode(http.StatusInternalServerError)
	}

	if response.Code >= http.StatusInternalServerError {
		zap.L().Error("HTTP 请求处理失败", zap.Error(err))
		response.SetMessage(http.StatusText(response.Code))
	}

	_ = GetContext(c).SendResponse(response.Code, response.Message)
}

// GetRequestURL 获取请求的原始URL（考虑nginx代理的情况）
// nginx反向代理的时候需要配置对应的Header，示例配置：
// proxy_set_header X-Original-URL $scheme://$http_host$request_uri;
// proxy_set_header X-Original-URI $request_uri;
// proxy_set_header X-Forwarded-Prefix /your-prefix;
// proxy_set_header X-Forwarded-Proto $scheme;
// proxy_set_header X-Forwarded-Host $host;
func GetRequestURL(c echo.Context) (u *url.URL, err error) {
	req := c.Request()

	// 完整原始地址优先，其次使用原始路径和查询参数
	originalURL := req.Header.Get("X-Original-URL")
	if originalURL == "" {
		originalURL = req.Header.Get("X-Original-URI")
	}

	if originalURL != "" {
		u, err = url.Parse(originalURL)
		if err != nil {
			err = fmt.Errorf("解析原始请求地址失败：%w", err)
			return
		}
	} else {
		// 如果没有 X-Original-URL，使用当前请求的 URL
		u = &url.URL{
			Path:        req.URL.Path,
			RawPath:     req.URL.RawPath,
			RawQuery:    req.URL.RawQuery,
			ForceQuery:  req.URL.ForceQuery,
			Fragment:    req.URL.Fragment,
			RawFragment: req.URL.RawFragment,
		}

		// 如果存在 X-Forwarded-Prefix，需要拼接前缀
		prefix := req.Header.Get("X-Forwarded-Prefix")
		if prefix != "" {
			u.RawPath = (&url.URL{Path: prefix}).EscapedPath() + u.EscapedPath()
			u.Path = prefix + u.Path
		}
	}

	// 补全原始地址中缺少的协议和主机
	if u.Scheme == "" {
		u.Scheme = req.Header.Get("X-Forwarded-Proto")
		if u.Scheme == "" {
			if req.TLS != nil {
				u.Scheme = "https"
			} else {
				u.Scheme = "http"
			}
		}
	}

	if u.Host == "" {
		u.Host = req.Header.Get("X-Forwarded-Host")
		if u.Host == "" {
			u.Host = req.Host
		}
	}

	return
}
