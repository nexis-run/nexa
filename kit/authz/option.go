// Copyright (C) aurservd. 2025-present.
//
// Created at 2025-10-21, by liasica

package authz

import (
	kr "github.com/go-kratos/kratos/v2/errors"
)

type option struct {
	errorHandler func(err error) error
}

var defaultOption = option{
	errorHandler: func(err error) error {
		if err == nil {
			return nil
		}

		return kr.FromError(err)
	},
}

type Option interface {
	apply(*option)
}

type optionFunc func(*option)

func (f optionFunc) apply(c *option) {
	f(c)
}

var _ = WithErrorHandler

// WithErrorHandler 设置错误处理
func WithErrorHandler(f func(err error) error) Option {
	return optionFunc(func(c *option) {
		c.errorHandler = f
	})
}
