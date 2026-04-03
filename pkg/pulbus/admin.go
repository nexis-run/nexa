// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import "github.com/apache/pulsar-client-go/pulsaradmin"

type Admin struct {
	pulsaradmin.Client
}

type AdminOption func(*pulsaradmin.Config)

func NewAdmin(webServiceURL string, opts ...AdminOption) (*Admin, error) {
	cfg := &pulsaradmin.Config{
		WebServiceURL: webServiceURL,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	admin, err := pulsaradmin.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Admin{Client: admin}, nil
}
