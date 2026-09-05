// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-06, by liasica

package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"nexis.run/nexa/kit"
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

	err = Setup(&configure.Logger{
		Name:   "test-log",
		Stdout: true,
	})
	require.NoError(t, err)

	zap.L().Info("test")
}

func TestSetupRejectsInvalidConfiguration(t *testing.T) {
	previous := zap.L()
	err := Setup(nil)
	require.ErrorIs(t, err, kit.ErrConfigInvalidLogger)

	err = Setup(&configure.Logger{Kafka: &configure.LoggerKafka{Disable: true}})
	require.ErrorIs(t, err, kit.ErrConfigInvalidLogger)
	require.Same(t, previous, zap.L())
}
