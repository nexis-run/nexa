// Copyright (C) nexa. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"github.com/labstack/echo/v4"
)

func ContextMiddleware(app string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			return next(NewContext(app, c))
		}
	}
}
