// Copyright (C) nexa. 2025-present.
//
// Created at 2025-02-17, by liasica

package gitlab

type Option interface {
	apply(h *Gitlab)
}

type optionFunc func(h *Gitlab)

func (f optionFunc) apply(h *Gitlab) {
	f(h)
}

func WithToken(token string) Option {
	return optionFunc(func(h *Gitlab) {
		h.AccessToken = token
	})
}

func WithBaseUrl(url string) Option {
	return optionFunc(func(h *Gitlab) {
		h.BaseUrl = url
	})
}
