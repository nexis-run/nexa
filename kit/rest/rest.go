// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// 防止静态检查工具误报
var _ = Run

type RouteHandler func(e *echo.Echo)

// Run 启动Rest服务
func Run(app, address string, r RouteHandler) (e *echo.Echo, ch chan error) {
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
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		_ = GetContext(c).SendResponse(http.StatusInternalServerError, err)
	}

	// 未找到错误
	echo.NotFoundHandler = func(c echo.Context) error {
		return GetContext(c).SendResponse(http.StatusNotFound)
	}

	// 请求方式错误
	echo.MethodNotAllowedHandler = func(c echo.Context) error {
		routerAllowMethods, ok := c.Get(echo.ContextKeyHeaderAllow).(string)
		if ok && routerAllowMethods != "" {
			c.Response().Header().Set(echo.HeaderAllow, routerAllowMethods)
		}

		return GetContext(c).SendResponse(http.StatusMethodNotAllowed)
	}

	// 设置全局中间件
	e.Use(
		ContextMiddleware(app),
		RecoverMiddleware(),
	)

	// 设置路由
	r(e)

	// 使用协程启动HTTP Rest服务器
	ch = make(chan error, 1)

	go func() {
		if err := e.Start(address); err != nil && !errors.Is(err, http.ErrServerClosed) {
			ch <- fmt.Errorf("HTTP Rest 服务启动失败: %w", err)
		}
	}()

	zap.L().Info(fmt.Sprintf("⇨ HTTP Rest server listening on %s", address))

	return
}

var _ = GetRequestURL

// GetRequestURL 获取请求的原始URL（考虑nginx代理的情况）
// nginx反向代理的时候需要配置对应的Header，示例配置：
// proxy_set_header X-Original-URL $scheme://$http_host$request_uri;
// proxy_set_header X-Original-URI $request_uri;
// proxy_set_header X-Forwarded-Prefix /your-prefix;
// proxy_set_header X-Forwarded-Proto $scheme;
// proxy_set_header X-Forwarded-Host $host;
func GetRequestURL(c echo.Context) (u *url.URL, err error) {
	req := c.Request()

	// 尝试从 X-Original-URL 获取原始请求URI
	originalURL := req.Header.Get("X-Original-URL")
	if originalURL != "" {
		u, err = url.Parse(originalURL)
		if err != nil {
			err = fmt.Errorf("解析 X-Original-URL 失败: %w", err)
			return
		}
	} else {
		// 如果没有 X-Original-URL，使用当前请求的 URL
		u = &url.URL{
			Path:     req.URL.Path,
			RawQuery: req.URL.RawQuery,
			Fragment: req.URL.Fragment,
		}

		// 如果存在 X-Forwarded-Prefix，需要拼接前缀
		prefix := req.Header.Get("X-Forwarded-Prefix")
		if prefix != "" {
			u.Path = prefix + u.Path
		}
	}

	// 设置协议（scheme）
	scheme := req.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if req.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	u.Scheme = scheme

	// 设置主机（host）
	host := req.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = req.Host
	}

	u.Host = host

	return
}
