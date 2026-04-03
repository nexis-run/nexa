// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-06, by liasica

package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"nexis.run/nexa/kit/configure"
)

func TestLogger(t *testing.T) {
	l, err := zap.NewProduction()
	require.NoError(t, err)
	l.Info("test")
	l.Named("xtest").Info("test")

	var ld *zap.Logger

	ld, err = zap.NewDevelopment()
	require.NoError(t, err)
	ld.Info("test")
	ld.Named("xtest").Info("test")

	Setup(&configure.Logger{
		Name:   "test-log",
		Stdout: true,
		Kafka: &configure.LoggerKafka{
			Topic: "applog",
			Brokers: []string{
				"10.10.10.200:32420",
				"10.10.10.200:32421",
				"10.10.10.200:32422",
			},
		},
	})

	zap.L().Info("KAFKA test")
}
