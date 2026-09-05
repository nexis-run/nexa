//go:build integration

// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-28, by liasica

package pulbus

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPulbus(t *testing.T) {
	bookie := os.Getenv("PULSAR_URL")
	adminURL := os.Getenv("PULSAR_ADMIN_URL")
	if bookie == "" || adminURL == "" {
		t.Skip("PULSAR_URL 和 PULSAR_ADMIN_URL 未设置")
	}

	bus, err := New(bookie, WithAdmin(adminURL))
	require.NoError(t, err)
	defer bus.Close()

	admin := bus.GetAdmin()
	require.NotNil(t, admin)

	var policies *utils.RetentionPolicies

	policies, err = admin.Namespaces().GetRetention(DefaultNamespace)
	require.NoError(t, err)
	require.NotNil(t, policies)
}

func TestConsume(t *testing.T) {
	bookie := os.Getenv("PULSAR_URL")
	if bookie == "" {
		t.Skip("PULSAR_URL 未设置")
	}

	bus, err := New(bookie)
	require.NoError(t, err)
	defer bus.Close()

	var consumer pulsar.Consumer
	topic := "nexa-integration-" + uuid.NewString()
	consumer, err = bus.client.Subscribe(pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: "nexa-integration-" + uuid.NewString(),
		Type:             pulsar.Shared,
	})
	require.NoError(t, err)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	err = bus.SendBytes(ctx, topic, []byte("test-message"))
	require.NoError(t, err)

	var msg pulsar.Message
	msg, err = consumer.Receive(ctx)
	require.NoError(t, err)
	require.Equal(t, "test-message", string(msg.Payload()))
}
