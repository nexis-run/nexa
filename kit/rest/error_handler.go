package rest

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	kr "github.com/go-kratos/kratos/v2/errors"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"google.golang.org/grpc/status"
)

func handleHTTPError(err error, c echo.Context) {
	if err == nil || c.Response().Committed {
		return
	}

	response := responseFromError(err)
	if response == nil {
		return
	}

	_ = GetContext(c).SendResponse(response.Code, response.Message)
}

// responseFromError 统一错误状态与消息，服务端错误只记录在日志中
func responseFromError(err error) *Response {
	if err == nil {
		return nil
	}

	value := reflect.ValueOf(err)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return nil
		}
	}

	response := NewResponse().SetCode(http.StatusInternalServerError)
	var responseErr *Error
	var httpErr *echo.HTTPError

	switch {
	case errors.As(err, &responseErr):
		if responseErr == nil {
			return nil
		}

		response.SetCode(responseErr.Code).SetMessage(responseErr.Message)
	case errors.As(err, &httpErr):
		if httpErr == nil {
			return nil
		}

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

	return response
}
