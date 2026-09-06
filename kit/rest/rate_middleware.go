// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-22, by liasica

package rest

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

type RateLimitOption func(config *middleware.RateLimiterConfig)

// RateLimitWithIdentifier 设置限流标识提取器
func RateLimitWithIdentifier(identifier middleware.Extractor) RateLimitOption {
	return func(config *middleware.RateLimiterConfig) {
		config.IdentifierExtractor = identifier
	}
}

// RateLimitWithMemoryStore 设置基于内存的限流存储
func RateLimitWithMemoryStore(limit, burst float64, expiresIn time.Duration) RateLimitOption {
	return func(config *middleware.RateLimiterConfig) {
		config.Store = middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: rate.Limit(limit), Burst: int(burst), ExpiresIn: expiresIn},
		)
	}
}

// RateLimitMiddleware 限流器中间件，默认基于 IP 限流，每秒允许 10 个请求，桶容量为 20
func RateLimitMiddleware(opts ...RateLimitOption) echo.MiddlewareFunc {
	config := middleware.RateLimiterConfig{
		Skipper: middleware.DefaultSkipper,
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{Rate: rate.Limit(10), Burst: 20, ExpiresIn: 0},
		),
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		ErrorHandler: func(echo.Context, error) error {
			return rateLimitExceeded()
		},
		DenyHandler: func(echo.Context, string, error) error {
			return rateLimitExceeded()
		},
	}

	for _, opt := range opts {
		opt(&config)
	}

	return middleware.RateLimiterWithConfig(config)
}

func rateLimitExceeded() error {
	return NewError(http.StatusTooManyRequests, "请求太频繁，请稍后再试")
}
