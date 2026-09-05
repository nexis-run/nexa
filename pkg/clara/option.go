// Copyright (C) nexa. 2025-present.
//
// Created at 2025-07-04, by liasica

package clara

import "time"

type Option interface {
	apply(*Writer)
}

type optionFunc func(*Writer)

func (f optionFunc) apply(c *Writer) {
	f(c)
}

func WithRetries(retries int) Option {
	return optionFunc(func(c *Writer) {
		c.retries = retries
	})
}

func WithTimeout(timeout time.Duration) Option {
	return optionFunc(func(c *Writer) {
		c.timeout = timeout
	})
}

func WithRetryInterval(interval time.Duration) Option {
	return optionFunc(func(c *Writer) {
		c.retryInterval = interval
	})
}
