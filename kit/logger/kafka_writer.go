// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-06, by liasica

package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"

	"nexis.run/nexa/pkg/clara"
)

const (
	kafkaWriteTimeout = 10 * time.Second
	kafkaChanBuffer   = 4096
	kafkaBatchSize    = 100
	kafkaMaxLogBytes  = 256 * 1024
	kafkaBufferBytes  = 16 * 1024 * 1024
)

var ErrKafkaLogTooLarge = errors.New("日志大小超过单条上限")

type kafkaLogEntry struct {
	payload    []byte
	syncResult chan error
}

type KafkaWriter struct {
	*clara.Writer

	ch         chan kafkaLogEntry
	loopDone   chan error
	closing    chan struct{}
	closeDone  chan struct{}
	stateMu    sync.RWMutex
	closeOnce  sync.Once
	closeErr   error
	droppedCnt atomic.Uint64
	buffered   atomic.Int64
}

func NewKafkaWriter(brokers []string, topic string) *KafkaWriter {
	w := &KafkaWriter{
		Writer:    clara.NewWriter(brokers, topic),
		ch:        make(chan kafkaLogEntry, kafkaChanBuffer),
		loopDone:  make(chan error, 1),
		closing:   make(chan struct{}),
		closeDone: make(chan struct{}),
	}

	go w.loop()

	return w
}

// Write 投递最多 256 KiB 的日志，待处理正文最多占用 16 MiB，容量耗尽时直接丢弃
func (w *KafkaWriter) Write(p []byte) (n int, err error) {
	w.stateMu.RLock()

	select {
	case <-w.closing:
		w.stateMu.RUnlock()

		err = io.ErrClosedPipe

		return
	default:
	}

	if len(p) > kafkaMaxLogBytes {
		w.droppedCnt.Add(1)
		w.stateMu.RUnlock()

		err = ErrKafkaLogTooLarge

		return
	}

	if !w.reserveBytes(len(p)) {
		w.stateMu.RUnlock()
		w.drop()

		n = len(p)

		return
	}

	// zap 会复用底层缓冲，入队前需要复制
	safeCopy := make([]byte, len(p))
	copy(safeCopy, p)

	dropped := false

	select {
	case w.ch <- kafkaLogEntry{payload: safeCopy}:
	default:
		w.buffered.Add(-int64(len(p)))

		dropped = true
	}
	w.stateMu.RUnlock()

	if dropped {
		w.drop()
	}

	n = len(p)

	return
}

// Dropped 返回超大日志及容量耗尽时丢弃的累计条数
func (w *KafkaWriter) Dropped() uint64 {
	return w.droppedCnt.Load()
}

func (w *KafkaWriter) reserveBytes(size int) bool {
	for {
		buffered := w.buffered.Load()
		if int64(size) > kafkaBufferBytes-buffered {
			return false
		}

		if w.buffered.CompareAndSwap(buffered, buffered+int64(size)) {
			return true
		}
	}
}

func (w *KafkaWriter) drop() {
	dropped := w.droppedCnt.Add(1)
	if dropped%1000 == 1 {
		_, _ = fmt.Fprintf(os.Stderr, "[KafkaWriter] 缓冲容量已满，已累计丢弃 %d 条日志\n", dropped)
	}
}

// loop 后台消费队列
func (w *KafkaWriter) loop() {
	var pendingErr error
	var closeErr error
	entries := make([]kafkaLogEntry, 0, kafkaBatchSize)
	messages := make([]kafka.Message, 0, kafkaBatchSize)

	for {
		entry, ok := <-w.ch
		if !ok {
			w.loopDone <- closeErr
			return
		}

		if entry.syncResult != nil {
			entry.syncResult <- pendingErr

			pendingErr = nil

			continue
		}

		entries = append(entries[:0], entry)
		var syncResult chan error
		closed := false

	collect:
		for len(entries) < cap(entries) {
			select {
			case entry, ok = <-w.ch:
				if !ok {
					closed = true
					break collect
				}

				if entry.syncResult != nil {
					syncResult = entry.syncResult
					break collect
				}

				entries = append(entries, entry)
			default:
				break collect
			}
		}

		messages = messages[:0]
		var batchBytes int64

		for _, item := range entries {
			messages = append(messages, kafka.Message{Value: item.payload})
			batchBytes += int64(len(item.payload))
		}

		err := w.sendBatch(context.Background(), messages)
		w.buffered.Add(-batchBytes)

		clear(entries[:cap(entries)])
		clear(messages[:cap(messages)])

		if err != nil {
			pendingErr = err
			closeErr = err
			_, _ = fmt.Fprintf(os.Stderr, "[KafkaWriter] 日志发送失败：%v\n", err)
		}

		if syncResult != nil {
			syncResult <- pendingErr

			pendingErr = nil
		}

		if closed {
			w.loopDone <- closeErr
			return
		}
	}
}

// sendBatch 隔离超大消息，保留同批中可正常投递的日志
func (w *KafkaWriter) sendBatch(ctx context.Context, messages []kafka.Message) (err error) {
	var rejectedErr error

	for len(messages) > 0 {
		err = w.SendMessagesAndWait(ctx, messages...)
		var oversized kafka.MessageTooLargeError

		if !errors.As(err, &oversized) {
			err = errors.Join(rejectedErr, err)
			return
		}

		err = errors.Join(rejectedErr, fmt.Errorf("日志大小为 %d 字节：%w", len(oversized.Message.Value), kafka.MessageSizeTooLarge))
		rejectedErr = err
		messages = oversized.Remaining
	}

	err = rejectedErr

	return
}

// Sync 最多等待 10 秒，超时不表示未完成的日志已被取消
func (w *KafkaWriter) Sync() error {
	ctx, cancel := context.WithTimeout(context.Background(), kafkaWriteTimeout)
	defer cancel()

	return w.SyncContext(ctx)
}

// SyncContext 等待此前已入队的日志完成发送尝试，不包含队列满时丢弃的日志
func (w *KafkaWriter) SyncContext(ctx context.Context) (err error) {
	err = ctx.Err()
	if err != nil {
		return
	}

	w.stateMu.RLock()

	select {
	case <-w.closing:
		w.stateMu.RUnlock()
		err = w.waitClose(ctx)

		return
	default:
	}

	result := make(chan error, 1)
	select {
	case w.ch <- kafkaLogEntry{syncResult: result}:
	case <-ctx.Done():
		w.stateMu.RUnlock()

		err = ctx.Err()

		return
	case <-w.closing:
		w.stateMu.RUnlock()
		err = w.waitClose(ctx)

		return
	}
	w.stateMu.RUnlock()

	select {
	case err = <-result:
	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}

// Close 停止接收日志并最多等待 10 秒，超时后后台继续释放资源
func (w *KafkaWriter) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), kafkaWriteTimeout)
	defer cancel()

	return w.CloseContext(ctx)
}

// CloseContext 停止接收日志并排空队列，返回生命周期内最后一次发送错误
// ctx 只限制调用方的等待时间
func (w *KafkaWriter) CloseContext(ctx context.Context) error {
	w.startClose()

	return w.waitClose(ctx)
}

func (w *KafkaWriter) startClose() {
	w.closeOnce.Do(func() {
		close(w.closing)

		go w.finishClose()
	})
}

func (w *KafkaWriter) finishClose() {
	w.stateMu.Lock()
	close(w.ch)
	w.stateMu.Unlock()

	sendErr := <-w.loopDone
	w.closeErr = errors.Join(sendErr, w.Writer.Close())
	close(w.closeDone)

	setupMu.Lock()
	delete(managedWriters, w)
	setupMu.Unlock()
}

func (w *KafkaWriter) waitClose(ctx context.Context) (err error) {
	select {
	case <-w.closeDone:
		err = w.closeErr
		return
	default:
	}

	select {
	case <-w.closeDone:
		err = w.closeErr
	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}
