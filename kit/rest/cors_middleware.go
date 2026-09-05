// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-06, by liasica

package rest

import (
	"slices"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type CORS struct {
	config middleware.CORSConfig
}

type CORSOption interface {
	apply(cors *CORS)
}

type corsOptionFunc func(cors *CORS)

func (f corsOptionFunc) apply(cors *CORS) {
	f(cors)
}

func CORSWithAllowOrigins(origins ...string) CORSOption {
	return corsOptionFunc(func(cors *CORS) {
		cors.config.AllowOrigins = slices.Clone(origins)
		if len(origins) == 0 {
			cors.config.AllowOrigins = []string{""}
		}
	})
}

func CORSWithAllowOriginFunc(f func(origin string) (bool, error)) CORSOption {
	return corsOptionFunc(func(cors *CORS) {
		cors.config.AllowOriginFunc = f
	})
}

func CORSWithAllowMethods(methods ...string) CORSOption {
	return corsOptionFunc(func(cors *CORS) {
		cors.config.AllowMethods = slices.Clone(methods)
		if len(methods) == 0 {
			cors.config.AllowMethods = []string{""}
		}
	})
}

func CORSWithAllowHeaders(headers ...string) CORSOption {
	return corsOptionFunc(func(cors *CORS) {
		cors.config.AllowHeaders = slices.Clone(headers)
		if len(headers) == 0 {
			cors.config.AllowHeaders = []string{""}
		}
	})
}

func CORSMiddlware(options ...CORSOption) echo.MiddlewareFunc {
	return CORSMiddleware(options...)
}

func CORSMiddleware(options ...CORSOption) echo.MiddlewareFunc {
	config := middleware.DefaultCORSConfig
	config.AllowOrigins = slices.Clone(config.AllowOrigins)
	config.AllowMethods = slices.Clone(config.AllowMethods)
	config.AllowHeaders = []string{HeaderContentType}
	config.ExposeHeaders = []string{HeaderContentType, HeaderDispositionType}

	cors := &CORS{
		config: config,
	}

	for _, option := range options {
		option.apply(cors)
	}

	return middleware.CORSWithConfig(cors.config)
}
