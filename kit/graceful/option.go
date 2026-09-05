// Copyright (C) nexa. 2025-present.
//
// Created at 2025-07-08, by liasica

package graceful

import "time"

type option struct {
	timeout time.Duration // 关闭超时时间，小于等于 0 时不设置截止时间
}

type Option func(*option)

func WithTimeout(d time.Duration) Option {
	return func(c *option) {
		c.timeout = d
	}
}
