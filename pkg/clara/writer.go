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
	DefaultBatchTimeout  = 10 * time.Millisecond
)

type Writer struct {
	writer *kafka.Writer

	retries       int
	retryInterval time.Duration
	timeout       time.Duration
}

// NewWriter 创建一个新的 Writer
func NewWriter(brokers []string, topic string, opts ...Option) *Writer {
	c := New(brokers)
	w := &Writer{
		writer: &kafka.Writer{
			Addr:                   kafka.TCP(c.brokers...),
			Topic:                  topic,
			AllowAutoTopicCreation: true,
			Async:                  false,
			Balancer:               &kafka.LeastBytes{},
			BatchSize:              100,
			BatchBytes:             1024 * 1024,
			BatchTimeout:           DefaultBatchTimeout,
			RequiredAcks:           kafka.RequireOne,
			Compression:            kafka.Snappy,
		},
		retries:       DefaultRetries,
		retryInterval: DefaultRetryInterval,
		timeout:       DefaultTimeout,
	}

	for _, opt := range opts {
		opt.apply(w)
	}

	if w.retries <= 0 {
		w.retries = 1
	}

	if w.retryInterval <= 0 {
		w.retryInterval = time.Nanosecond
	}

	if w.timeout <= 0 {
		w.timeout = DefaultTimeout
	}

	w.writer.MaxAttempts = w.retries
	w.writer.WriteBackoffMin = w.retryInterval
	w.writer.WriteBackoffMax = w.retryInterval
	w.writer.ReadTimeout = w.timeout
	w.writer.WriteTimeout = w.timeout

	return w
}

// With 自定义 writer 配置
func (w *Writer) With(fn func(writer *kafka.Writer)) *Writer {
	fn(w.writer)
	return w
}

// SendMessages 提交一次消息，分区重试由底层 writer 负责
// 超时或取消后消息仍可能被投递，调用方重发前需要考虑重复处理
func (w *Writer) SendMessages(ctx context.Context, messages ...kafka.Message) (err error) {
	err = ctx.Err()
	if err != nil {
		return
	}

	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(ctx, w.timeout)
	defer cancel()

	err = w.writer.WriteMessages(ctx, messages...)

	return
}

// SendMessagesAndWait 等待底层同步发送完成，不额外添加等待超时
// ctx 取消后结果仍可能不确定，不支持异步 writer
func (w *Writer) SendMessagesAndWait(ctx context.Context, messages ...kafka.Message) (err error) {
	err = ctx.Err()
	if err != nil {
		return
	}

	if w.writer.Async {
		err = errors.New("同步等待不支持异步 writer")
		return
	}

	err = w.writer.WriteMessages(ctx, messages...)

	return
}

// Close 关闭writer
func (w *Writer) Close() error {
	return w.writer.Close()
}
