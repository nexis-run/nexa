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

type Reader struct {
	groupID string
	reader  *kafka.Reader
}

type MessageListener func(message kafka.Message, err error) error

// NewReader 创建一个新的Kafka Reader
func NewReader(brokers []string, topic, groupID string) *Reader {
	c := New(brokers)

	return &Reader{
		groupID: groupID,
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  c.brokers,
			Topic:    topic,
			GroupID:  groupID,
			MaxBytes: 10e6, // 10MB
			// https://github.com/segmentio/kafka-go/issues/800#issuecomment-981855523
			WatchPartitionChanges:  true,
			PartitionWatchInterval: time.Second * 5,
		}),
	}
}

// With 自定义reader配置
func (r *Reader) With(fn func(reader *kafka.Reader)) *Reader {
	fn(r.reader)
	return r
}

// Listen 监听消息回调
func (r *Reader) Listen(ctx context.Context, cb MessageListener) error {
	if cb == nil {
		return errors.New("消息处理函数不能为空")
	}

	// 接收消息
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		message, err := r.reader.FetchMessage(ctx)
		if err != nil {
			return errors.Join(err, cb(message, err))
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		err = cb(message, nil)
		if err != nil {
			return err
		}

		// 处理成功后提交 offset
		if r.groupID != "" {
			err = r.reader.CommitMessages(ctx, message)
			if err != nil {
				return err
			}
		}
	}
}

// Reader 获取底层kafka Reader实例
func (r *Reader) Reader() *kafka.Reader {
	return r.reader
}

// Close 关闭reader
func (r *Reader) Close() error {
	return r.reader.Close()
}
