// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"context"
	"errors"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ProducerOption func(*pulsar.ProducerMessage)

// WithProducerKey 设置消息 Key
// 的作用:
// 1. 分区路由: 相同 Key 的消息发送到同一分区，保证顺序性
// 2. 消息去重: 启用去重后，根据 Key 判断重复消息
// 3. 压缩支持: Topic Compaction 时，相同 Key 只保留最新消息
// 4. Key_Shared 模式: 相同 Key 的消息发送到同一 Consumer 实例
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

// 生产日志记录
func (producer *Producer) log(level zapcore.Level, message string, data pulsar.Message) {
	b, _ := sonic.Marshal(data)
	zap.L().Log(level, "[Pulsar Producer] "+message, zap.ByteString("message", b), zap.String("topic", producer.Topic()))
}

// getProducer 获取 Producer
func (bus *Pulbus) getProducer(topic string, opts pulsar.ProducerOptions) (pulsar.Producer, error) {
	// 尝试从缓存中获取
	if p, ok := bus.producers.Load(topic); ok {
		return p.(pulsar.Producer), nil
	}

	// 不存在则创建新的 producer
	opts.Topic = topic

	producer, err := bus.client.CreateProducer(opts)
	if err != nil {
		return nil, err
	}

	// 存入缓存
	bus.producers.Store(topic, producer)

	return producer, nil
}

// Send 发送消息到指定 Topic
func (bus *Pulbus) Send(ctx context.Context, topic string, messageOpts ...ProducerOption) error {
	producer, err := bus.getProducer(topic, pulsar.ProducerOptions{})
	if err != nil {
		return err
	}

	msg := &pulsar.ProducerMessage{}

	// 应用自定义选项
	for _, opt := range messageOpts {
		opt(msg)
	}

	// 判定消息内容是否为空（Payload 和 Value 至少需要一个）
	if msg.Payload == nil && msg.Value == nil {
		return errors.New("消息内容不能为空")
	}

	_, err = producer.Send(ctx, msg)

	return err
}

// SendBytes 发送消息到指定 Topic
func (bus *Pulbus) SendBytes(ctx context.Context, topic string, b []byte, messageOpts ...ProducerOption) error {
	return bus.Send(ctx, topic, append(messageOpts, WithPayload(b))...)
}
