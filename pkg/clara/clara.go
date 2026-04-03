// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-09, by liasica

package clara

import (
	"strings"

	cmap "github.com/orcaman/concurrent-map/v2"
)

var instances = cmap.New[*Clara]()

type Clara struct {
	brokers []string
	writers cmap.ConcurrentMap[string, *Writer]
}

func New(brokers []string) *Clara {
	key := strings.Join(brokers, ",")
	if instance, exists := instances.Get(key); exists {
		return instance
	}

	c := &Clara{
		brokers: brokers,
		writers: cmap.New[*Writer](),
	}

	instances.Set(key, c)

	return c
}
