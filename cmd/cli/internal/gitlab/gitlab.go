// Copyright (C) nexa. 2025-present.
//
// Created at 2025-02-17, by liasica

package gitlab

import gitlab "gitlab.com/gitlab-org/api/client-go"

const (
	DefaultRepository = "nexis/nexa"
	DefaultBranch     = "master"
)

type Gitlab struct {
	BaseUrl     string
	AccessToken string

	client *gitlab.Client
}

func New(token string, options ...Option) (g *Gitlab, err error) {
	g = &Gitlab{
		BaseUrl:     `https://gitlab.liasica.com`,
		AccessToken: token,
	}
	for _, o := range options {
		o.apply(g)
	}

	g.client, err = gitlab.NewClient(g.AccessToken, gitlab.WithBaseURL(g.BaseUrl))
	if err != nil {
		return nil, err
	}

	return
}
