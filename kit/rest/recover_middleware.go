// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func RecoverMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := GetContext(c)

			defer func() {
				if r := recover(); r != nil {
					switch v := r.(type) {
					case *Error:
						_ = ctx.SendResponse(v.Code, v.Message)
					default:
						err := fmt.Errorf("%v", v)
						zap.L().Error("捕获HTTP未处理崩溃", zap.Error(err), zap.Stack("stack"))

						_ = ctx.SendResponse(http.StatusInternalServerError, "Internal Server Error")
					}
				}
			}()

			return next(ctx)
		}
	}
}
