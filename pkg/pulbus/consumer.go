// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"context"
	"errors"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"go.uber.org/zap"
)

const (
	defaultConsumerChannelSize = 200
)

var (
	ErrConsumerChannelClosed = errors.New("消费者消息通道已关闭")
	ErrInvalidChannelSize    = errors.New("消费者消息通道大小不能小于 0")
	ErrNilMessageHandler     = errors.New("消息处理函数不能为空")
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
	key       ConsumerKey
	closeOnce sync.Once

	pulsar.Consumer
}

// Close 幂等关闭 Consumer
func (consumer *Consumer) Close() {
	consumer.closeOnce.Do(consumer.Consumer.Close)
}

// 消费日志记录
func (consumer *Consumer) log(message string, err error) {
	zap.L().Warn("[Pulsar Consumer] "+message, zap.Error(err), zap.String("topic", consumer.key.Topic), zap.String("subscription", consumer.key.Subscription))
}

// getConsumer 创建并跟踪 Consumer
func (bus *Pulbus) getConsumer(topic, subscription string, opts pulsar.ConsumerOptions) (*Consumer, error) {
	bus.lifecycleMu.Lock()
	if bus.closed {
		bus.lifecycleMu.Unlock()
		return nil, ErrClosed
	}
	bus.creating.Add(1)
	bus.lifecycleMu.Unlock()

	defer bus.creating.Done()

	key := ConsumerKey{Topic: topic, Subscription: subscription}

	consumer := &Consumer{
		key: key,
	}

	opts.Topic = topic
	opts.SubscriptionName = subscription

	var err error

	consumer.Consumer, err = bus.client.Subscribe(opts)
	if err != nil {
		return nil, err
	}

	bus.lifecycleMu.Lock()
	if bus.closed {
		bus.lifecycleMu.Unlock()
		consumer.Close()

		return nil, ErrClosed
	}

	if bus.consumers == nil {
		bus.consumers = make(map[*Consumer]struct{})
	}

	bus.consumers[consumer] = struct{}{}
	bus.lifecycleMu.Unlock()

	return consumer, nil
}

func (bus *Pulbus) closeConsumer(consumer *Consumer) {
	bus.lifecycleMu.Lock()
	delete(bus.consumers, consumer)
	bus.lifecycleMu.Unlock()
	consumer.Close()
}

// 处理消息
func (consumer *Consumer) handleMessage(msg pulsar.Message, handler MessageHandler) {
	// 如果返回失败则 nack 该条消息并继续接收下一条消息
	err := handler(msg)
	if err != nil {
		// 处理失败，nack 消息
		consumer.Nack(msg)
		consumer.log("消息处理失败，Nack 消息", err)

		return
	}

	// 处理成功，ack 消息
	err = consumer.Ack(msg)
	if err != nil {
		consumer.log("Ack 失败", err)
	}
}

// ConsumeWithLoop 阻塞消费消息
func (bus *Pulbus) ConsumeWithLoop(ctx context.Context, topic, subscription string, handler MessageHandler) error {
	return bus.Consume(ctx, topic, subscription, handler)
}

// Consume 使用 channel 阻塞消费消息
func (bus *Pulbus) Consume(ctx context.Context, topic, subscription string, handler MessageHandler, opts ...ConsumerOption) error {
	if handler == nil {
		return ErrNilMessageHandler
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	options := &ConsumerOptions{
		channelSize: defaultConsumerChannelSize,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(options)
		}
	}

	if options.channelSize < 0 {
		return ErrInvalidChannelSize
	}

	messageChan := make(chan pulsar.ConsumerMessage, options.channelSize)

	consumer, err := bus.getConsumer(topic, subscription, pulsar.ConsumerOptions{
		Type:           pulsar.Shared,
		MessageChannel: messageChan,
	})
	if err != nil {
		return err
	}

	defer bus.closeConsumer(consumer)

	// 从 channel 读取消息
	for {
		select {
		case <-bus.closing:
			return ErrClosed
		case <-ctx.Done():
			return ctx.Err()
		case cm, ok := <-messageChan:
			if !ok {
				return ErrConsumerChannelClosed
			}

			if ctx.Err() != nil {
				return ctx.Err()
			}

			select {
			case <-bus.closing:
				return ErrClosed
			default:
			}

			consumer.handleMessage(cm.Message, handler)
		}
	}
}
