// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-28, by liasica

package pulbus

import (
	"context"
	"fmt"
	"testing"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
	"github.com/stretchr/testify/require"
)

func TestPulbus(t *testing.T) {
	bus, err := New("pulsar://10.10.10.220:36650", WithAdmin("http://10.10.10.220:36651"))
	require.NoError(t, err)

	admin := bus.GetAdmin()
	require.NotNil(t, admin)

	var policies *utils.RetentionPolicies

	policies, err = admin.Namespaces().GetRetention(DefaultNamespace)
	require.NoError(t, err)
	fmt.Printf("Retention policies for namespace %s: %+v\n", DefaultNamespace, policies)
}

func TestConsume(t *testing.T) {
	bus, err := New("pulsar://10.10.10.220:36650")
	require.NoError(t, err)

	defer bus.client.Close()

	var consumer pulsar.Consumer

	consumer, err = bus.client.Subscribe(pulsar.ConsumerOptions{
		Topic:            "test-Topic",
		SubscriptionName: "test-sub",
		Type:             pulsar.Shared,
	})
	require.NoError(t, err)

	var msg pulsar.Message

	msg, err = consumer.Receive(context.Background())
	require.NoError(t, err)
	fmt.Printf("Received message msgId: %#v -- content: '%s'\n", msg.ID(), string(msg.Payload()))
}
