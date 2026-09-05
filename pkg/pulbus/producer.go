// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"context"
	"errors"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
)

type ProducerOption func(*pulsar.ProducerMessage)

// WithProducerKey 设置消息 Key
// 消息路由与压缩行为：
// 1. 分区路由：相同 Key 的消息发送到同一分区，保证顺序性
// 2. Topic Compaction：相同 Key 只保留最新消息
// 3. Key_Shared 模式：相同 Key 的消息发送到同一 Consumer 实例
//
// 使用示例:
//
//	bus.Send(ctx, "orders", WithProducerKey("user:123"), WithPayload(data))
func WithProducerKey(key string) ProducerOption {
	return func(message *pulsar.ProducerMessage) {
		message.Key = key
	}
}

// WithPayload 设置消息内容
func WithPayload(payload []byte) ProducerOption {
	return func(message *pulsar.ProducerMessage) {
		message.Payload = payload
	}
}

// WithProducerDeliverAfter 设置延迟投递时间
func WithProducerDeliverAfter(d time.Duration) ProducerOption {
	return func(message *pulsar.ProducerMessage) {
		message.DeliverAfter = d
	}
}

// WithSequenceID 手动指定序列号（用于消息去重）
//
// 注意: Pulsar 去重默认未启用,需要先配置:
//
//	bin/pulsar-admin namespaces set-deduplication --enable tenant/namespace
//
// 使用示例:
//
//	bus.Send(ctx, "orders",
//	    WithSequenceID(123),
//	    WithPayload(data),
//	)
func WithSequenceID(seqID int64) ProducerOption {
	return func(message *pulsar.ProducerMessage) {
		message.SequenceID = &seqID
	}
}

type Producer struct {
	pulsar.Producer
}

type producerEntry struct {
	ready    chan struct{}
	producer pulsar.Producer
	err      error
}

// getProducer 合并同一 Topic 的并发创建，不阻塞其他 Topic
func (bus *Pulbus) getProducer(ctx context.Context, topic string) (pulsar.Producer, error) {
	bus.lifecycleMu.Lock()
	if bus.closed {
		bus.lifecycleMu.Unlock()
		return nil, ErrClosed
	}

	entry, exists := bus.producers[topic]
	if !exists {
		entry = &producerEntry{ready: make(chan struct{})}

		if bus.producers == nil {
			bus.producers = make(map[string]*producerEntry)
		}

		bus.producers[topic] = entry
		bus.creating.Add(1)
	}
	bus.lifecycleMu.Unlock()

	if !exists {
		go bus.createProducer(topic, entry)
	}

	select {
	case <-entry.ready:
		return entry.producer, entry.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-bus.closing:
		return nil, ErrClosed
	}
}

func (bus *Pulbus) createProducer(topic string, entry *producerEntry) {
	defer bus.creating.Done()

	producer, err := bus.client.CreateProducer(pulsar.ProducerOptions{Topic: topic})

	bus.lifecycleMu.Lock()
	if err == nil && bus.closed {
		err = ErrClosed
	}

	entry.producer, entry.err = producer, err
	if err != nil {
		delete(bus.producers, topic)

		entry.producer = nil
	}

	close(entry.ready)
	bus.lifecycleMu.Unlock()

	if err != nil && producer != nil {
		producer.Close()
	}
}

// Send 发送消息到指定 Topic
func (bus *Pulbus) Send(ctx context.Context, topic string, messageOpts ...ProducerOption) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	msg := &pulsar.ProducerMessage{}

	// 应用自定义选项
	for _, opt := range messageOpts {
		if opt != nil {
			opt(msg)
		}
	}

	// 判定消息内容是否为空（Payload 和 Value 至少需要一个）
	if msg.Payload == nil && msg.Value == nil {
		return errors.New("消息内容不能为空")
	}

	producer, err := bus.getProducer(ctx, topic)
	if err != nil {
		return err
	}

	_, err = producer.Send(ctx, msg)

	return err
}

// SendBytes 发送消息到指定 Topic
func (bus *Pulbus) SendBytes(ctx context.Context, topic string, b []byte, messageOpts ...ProducerOption) error {
	options := make([]ProducerOption, len(messageOpts)+1)
	copy(options, messageOpts)
	options[len(messageOpts)] = WithPayload(b)

	return bus.Send(ctx, topic, options...)
}
