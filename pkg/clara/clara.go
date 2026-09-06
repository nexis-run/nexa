// Copyright (C) nexa. 2024-present.
//
// Created at 2024-12-09, by liasica

package clara

import "slices"

type Clara struct {
	brokers []string
}

func New(brokers []string) *Clara {
	return &Clara{
		brokers: slices.Clone(brokers),
	}
}
