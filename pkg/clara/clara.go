// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-09, by liasica

package clara

type Clara struct {
	brokers []string
}

func New(brokers []string) *Clara {
	return &Clara{
		brokers: append([]string(nil), brokers...),
	}
}
