// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"context"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/bytedance/sonic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	defaultConsumerChannelSize = 200
)

// MessageHandler 消息处理函数类型
type MessageHandler func(msg pulsar.Message) error

// ConsumerKey 生成 consumer 的唯一标识
type ConsumerKey struct {
	Topic        string
	Subscription string
}

type ConsumerOptions struct {
	channelSize int
}

type ConsumerOption func(*ConsumerOptions)

// WithConsumerChannelSize 设置消费 channel 缓冲大小
func WithConsumerChannelSize(size int) ConsumerOption {
	return func(o *ConsumerOptions) {
		o.channelSize = size
	}
}

type Consumer struct {
	key ConsumerKey

	pulsar.Consumer
}

// 消费日志记录
func (consumer *Consumer) log(level zapcore.Level, message string, data pulsar.Message) {
	b, _ := sonic.Marshal(data)
	zap.L().Log(level, "[Pulsar Consumer] "+message, zap.ByteString("message", b), zap.String("topic", consumer.key.Topic), zap.String("subscription", consumer.key.Subscription))
}

// getConsumer 获取 Consumer（使用 LoadOrStore 避免并发创建）
func (bus *Pulbus) getConsumer(topic, subscription string, opts pulsar.ConsumerOptions) (*Consumer, error) {
	key := ConsumerKey{Topic: topic, Subscription: subscription}

	// 尝试从缓存中获取
	if c, ok := bus.consumers.Load(key); ok {
		return c.(*Consumer), nil
	}

	consumer := &Consumer{
		key: key,
	}

	// 不存在则创建新的 consumer
	opts.Topic = topic
	opts.SubscriptionName = subscription

	var err error

	consumer.Consumer, err = bus.client.Subscribe(opts)
	if err != nil {
		return nil, err
	}

	// 使用 LoadOrStore 原子操作，避免并发创建多个 consumer
	actual, loaded := bus.consumers.LoadOrStore(key, consumer)
	if loaded {
		// 已存在其他 goroutine 创建的 consumer，关闭当前的
		consumer.Close()
		return actual.(*Consumer), nil
	}

	return consumer, nil
}

// 处理消息
func (consumer *Consumer) handleMessage(msg pulsar.Message, handler MessageHandler) {
	// 如果返回失败则 nack 该条消息并继续接收下一条消息
	err := handler(msg)
	if err != nil {
		// 处理失败，nack 消息
		consumer.Nack(msg)
		consumer.log(zapcore.WarnLevel, "消息处理失败，Nack 消息", msg)

		return
	}

	// 处理成功，ack 消息
	err = consumer.Ack(msg)
	if err != nil {
		consumer.log(zapcore.WarnLevel, "Ack 失败", msg)
	}
}

// ConsumeWithLoop 阻塞消费消息
func (bus *Pulbus) ConsumeWithLoop(ctx context.Context, topic, subscription string, handler MessageHandler) error {
	// 使用缓存的 consumer
	consumer, err := bus.getConsumer(topic, subscription, pulsar.ConsumerOptions{
		Type: pulsar.Shared,
	})
	if err != nil {
		return err
	}

	// 持续接收消息
	var msg pulsar.Message

	for {
		msg, err = consumer.Receive(ctx)
		if err != nil {
			return err
		}

		consumer.handleMessage(msg, handler)
	}
}

// Consume 使用 channel 阻塞消费消息
func (bus *Pulbus) Consume(ctx context.Context, topic, subscription string, handler MessageHandler, opts ...ConsumerOption) error {
	options := &ConsumerOptions{
		channelSize: defaultConsumerChannelSize,
	}

	for _, opt := range opts {
		opt(options)
	}

	messageChan := make(chan pulsar.ConsumerMessage, options.channelSize)

	// 使用缓存的 consumer
	consumer, err := bus.getConsumer(topic, subscription, pulsar.ConsumerOptions{
		Type:           pulsar.Shared,
		MessageChannel: messageChan,
	})
	if err != nil {
		return err
	}

	// ctx 取消时关闭 consumer 并从缓存中移除，避免 goroutine 和 channel 泄露
	defer func() {
		consumer.Close()
		bus.consumers.Delete(consumer.key)
	}()

	// 从 channel 读取消息
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case cm := <-messageChan:
			consumer.handleMessage(cm.Message, handler)
		}
	}
}
