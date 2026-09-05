// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/apache/pulsar-client-go/pulsaradmin"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/admin/auth"
)

type Admin struct {
	pulsaradmin.Client
}

type AdminOption func(*pulsaradmin.Config)

func NewAdmin(webServiceURL string, opts ...AdminOption) (*Admin, error) {
	cfg := &pulsaradmin.Config{
		WebServiceURL: webServiceURL,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	cfg.WebServiceURL = strings.TrimSpace(cfg.WebServiceURL)

	endpoint, err := url.Parse(cfg.WebServiceURL)
	if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Hostname() == "" ||
		endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, errors.New("管理地址必须是完整的 HTTP 或 HTTPS URL，且不能包含查询参数或片段")
	}

	switch cfg.AuthPlugin {
	case "", auth.TLSPluginName, auth.TLSPluginShortName, auth.TokenPluginName, auth.TokePluginShortName, auth.OAuth2PluginName, auth.OAuth2PluginShortName:
	default:
		return nil, fmt.Errorf("不支持的 Pulsar Admin 认证插件：%s", cfg.AuthPlugin)
	}

	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return nil, errors.New("客户端 TLS 证书和私钥必须同时配置")
	}

	var admin pulsaradmin.Client

	admin, err = pulsaradmin.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Admin{Client: admin}, nil
}
