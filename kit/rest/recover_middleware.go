// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func RecoverMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c = ensureContext(c)

			defer func() {
				if r := recover(); r != nil {
					if r == http.ErrAbortHandler {
						panic(r)
					}

					var err error

					switch v := r.(type) {
					case error:
						err = v
					default:
						err = fmt.Errorf("%v", v)
					}

					var responseErr *Error
					if !errors.As(err, &responseErr) {
						zap.L().Error("捕获 HTTP 未处理崩溃", zap.Error(err), zap.Stack("stack"))
					}

					handleHTTPError(err, c)
				}
			}()

			return next(c)
		}
	}
}
