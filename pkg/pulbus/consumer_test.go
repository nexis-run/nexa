// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"context"
	"testing"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/require"
)

func TestConsumerOption(t *testing.T) {
	options := &ConsumerOptions{}

	WithConsumerChannelSize(100)(options)

	require.Equal(t, options.channelSize, 100)
}

func TestConsumeValidatesOptions(t *testing.T) {
	bus := &Pulbus{}

	err := bus.Consume(context.Background(), "topic", "subscription", nil)
	require.ErrorIs(t, err, ErrNilMessageHandler)

	err = bus.Consume(
		context.Background(),
		"topic",
		"subscription",
		func(pulsar.Message) error { return nil },
		WithConsumerChannelSize(-1),
	)
	require.ErrorIs(t, err, ErrInvalidChannelSize)
}
