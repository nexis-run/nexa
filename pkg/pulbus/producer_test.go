// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/stretchr/testify/require"
)

// TestProducerOptions 测试所有 ProducerOption
func TestProducerOptions(t *testing.T) {
	key := "test-key"
	payload := []byte("test")
	duration := 10 * time.Second
	seqID := int64(123)

	// 确保所有选项函数可以正常使用
	opts := []ProducerOption{
		WithProducerKey(key),
		WithPayload(payload),
		WithProducerDeliverAfter(duration),
		WithSequenceID(seqID),
	}

	msg := &pulsar.ProducerMessage{}

	for _, opt := range opts {
		opt(msg)
	}

	require.Equal(t, msg.Key, key)
	require.Equal(t, msg.Payload, payload)
	require.Equal(t, msg.DeliverAfter, duration)
	require.NotNil(t, msg.SequenceID)
	require.Equal(t, *msg.SequenceID, seqID)
}
