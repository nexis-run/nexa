// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-09, by liasica

package clara

import (
	"context"
	"errors"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	DefaultRetries       = 3                      // 默认重试次数
	DefaultTimeout       = 3 * time.Second        // 默认超时时间
	DefaultRetryInterval = 250 * time.Millisecond // 默认重试间隔
)

type Writer struct {
	writer *kafka.Writer

	retries       int
	retryInterval time.Duration
	timeout       time.Duration
}

var _ = NewWriter

// NewWriter 创建一个新的 Writer
func NewWriter(brokers []string, topic string, opts ...Option) *Writer {
	c := New(brokers)

	w, exists := c.writers.Get(topic)
	if exists {
		return w
	}

	w = &Writer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(c.brokers...),
			Topic:                  topic,
			AllowAutoTopicCreation: true,                // 自动创建topic
			Async:                  true,                // 异步
			Balancer:               &kafka.LeastBytes{}, // 选择分区策略，这里使用最小字节策略（保持）
			BatchSize:              100,                 // 设置批次大小，以消息数量为单位（选填）
			BatchBytes:             1024 * 1024,         // 设置批次字节大小上限（选填）
			BatchTimeout:           1 * time.Second,     // 批次超时时间，触发批量发送的超时机制（选填）
			RequiredAcks:           kafka.RequireOne,    // 设置应答级别，仅需一个副本确认，平衡可靠性和性能（保持，默认kafka.RequireNone）
			Compression:            kafka.Snappy,        // 使用Snappy压缩以减少网络传输量（选填）
		},
		retries:       DefaultRetries,
		retryInterval: DefaultRetryInterval,
		timeout:       DefaultTimeout,
	}

	for _, opt := range opts {
		opt.apply(w)
	}

	c.writers.Set(topic, w)

	return w
}

// With 自定义reader配置
func (w *Writer) With(fn func(reader *kafka.Writer)) *Writer {
	fn(w.writer)
	return w
}

// SendMessages 发送消息到Kafka
func (w *Writer) SendMessages(ctx context.Context, messages ...kafka.Message) (err error) {
	for i := 0; i < w.retries; i++ {
		err = w.writeMessagesWithTimeout(ctx, messages...)
		if errors.Is(err, kafka.LeaderNotAvailable) || errors.Is(err, kafka.UnknownTopicOrPartition) || errors.Is(err, context.DeadlineExceeded) {
			time.Sleep(w.retryInterval)
			continue
		}

		return
	}

	return
}

// writeMessagesWithTimeout 写入消息，带有超时控制
func (w *Writer) writeMessagesWithTimeout(ctx context.Context, messages ...kafka.Message) (err error) {
	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(ctx, w.timeout)
	defer cancel()

	return w.writer.WriteMessages(ctx, messages...)
}

// Close 关闭writer
func (w *Writer) Close() error {
	return w.writer.Close()
}
