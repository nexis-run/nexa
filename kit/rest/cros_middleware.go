// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-06, by liasica

package rest

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// cors
var corsConfig = middleware.DefaultCORSConfig

func init() {
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, []string{
		HeaderContentType,
	}...)
	corsConfig.ExposeHeaders = append(corsConfig.ExposeHeaders, []string{
		HeaderContentType,
		HeaderDispositionType,
	}...)
}

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
		cors.config.AllowOrigins = append(cors.config.AllowOrigins, origins...)
	})
}

func CORSWithAllowOriginFunc(f func(origin string) (bool, error)) CORSOption {
	return corsOptionFunc(func(cors *CORS) {
		cors.config.AllowOriginFunc = f
	})
}

func CORSWithAllowMethods(methods ...string) CORSOption {
	return corsOptionFunc(func(cors *CORS) {
		cors.config.AllowMethods = append(cors.config.AllowMethods, methods...)
	})
}

func CORSWithAllowHeaders(headers ...string) CORSOption {
	return corsOptionFunc(func(cors *CORS) {
		cors.config.AllowHeaders = append(cors.config.AllowHeaders, headers...)
	})
}

func CORSMiddlware(options ...CORSOption) echo.MiddlewareFunc {
	cors := &CORS{
		config: corsConfig,
	}

	for _, option := range options {
		option.apply(cors)
	}

	return middleware.CORSWithConfig(cors.config)
}
