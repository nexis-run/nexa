// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-06, by liasica

package logger

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"

	"nexis.run/nexa/pkg/clara"
)

const (
	kafkaWriteTimeout = 10 * time.Second
	kafkaChanBuffer   = 4096 // 异步队列容量, 满后采用丢弃策略避免业务侧阻塞
)

type KafkaWriter struct {
	*clara.Writer

	ch         chan []byte
	droppedCnt atomic.Uint64 // 队列满导致丢弃的日志条数
}

func NewKafkaWriter(brokers []string, topic string) *KafkaWriter {
	w := &KafkaWriter{
		Writer: clara.NewWriter(brokers, topic),
		ch:     make(chan []byte, kafkaChanBuffer),
	}

	go w.loop()

	return w
}

// Write 投递日志到异步发送队列, 队列满时直接丢弃, 避免阻塞业务路径
func (w *KafkaWriter) Write(p []byte) (n int, err error) {
	// 必须 copy, zap 复用同一份底层缓冲
	safeCopy := make([]byte, len(p))
	copy(safeCopy, p)

	select {
	case w.ch <- safeCopy:
	default:
		dropped := w.droppedCnt.Add(1)

		// 每累计 1000 条丢弃日志才向 stderr 提示一次, 防止刷屏
		if dropped%1000 == 1 {
			_, _ = fmt.Fprintf(os.Stderr, "[KafkaWriter] 队列已满, 已累计丢弃 %d 条日志\n", dropped)
		}
	}

	return len(p), nil
}

// loop 后台消费队列, 串行调用 SendMessages
func (w *KafkaWriter) loop() {
	for payload := range w.ch {
		ctx, cancel := context.WithTimeout(context.Background(), kafkaWriteTimeout)

		err := w.SendMessages(ctx, kafka.Message{Value: payload})

		cancel()

		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "[KafkaWriter] 日志发送失败: %v\n", err)
		}
	}
}

func (w *KafkaWriter) Sync() error {
	return nil
}
