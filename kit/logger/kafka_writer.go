// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-06, by liasica

package logger

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/segmentio/kafka-go"

	"nexis.run/nexa/pkg/clara"
)

const kafkaWriteTimeout = 10 * time.Second

type KafkaWriter struct {
	*clara.Writer
}

func NewKafkaWriter(brokers []string, topic string) *KafkaWriter {
	return &KafkaWriter{
		Writer: clara.NewWriter(brokers, topic),
	}
}

func (w *KafkaWriter) Write(p []byte) (n int, err error) {
	// 创建一个副本以避免数据竞争
	safeCopy := make([]byte, len(p))
	copy(safeCopy, p)

	ctx, cancel := context.WithTimeout(context.Background(), kafkaWriteTimeout)
	defer cancel()

	err = w.SendMessages(ctx, kafka.Message{
		Value: safeCopy,
	})
	if err != nil {
		// 日志发送失败时输出到 stderr，避免关键日志静默丢失
		_, _ = fmt.Fprintf(os.Stderr, "[KafkaWriter] 日志发送失败: %v\n", err)
		return
	}
	n = len(p)
	return
}

func (w *KafkaWriter) Sync() error {
	return nil
}
