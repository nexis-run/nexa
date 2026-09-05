// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-28, by liasica

package pulbus

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"go.uber.org/zap"
)

var ErrClosed = errors.New("消息客户端已关闭")

type Pulbus struct {
	client        pulsar.Client
	admin         *Admin
	clientOptions pulsar.ClientOptions
	initErr       error

	producers map[string]*producerEntry
	consumers map[*Consumer]struct{}

	lifecycleMu sync.Mutex
	creating    sync.WaitGroup
	closeOnce   sync.Once
	closing     chan struct{}
	closeDone   chan struct{}
	closed      bool
}

// Option Pulbus 配置选项
type Option func(bus *Pulbus)

// WithAdmin 配置 Pulsar Admin 客户端
func WithAdmin(webServiceURL string, opts ...AdminOption) Option {
	return func(bus *Pulbus) {
		admin, err := NewAdmin(webServiceURL, opts...)
		if err != nil {
			bus.initErr = errors.Join(bus.initErr, fmt.Errorf("创建 Pulsar Admin 失败：%w", err))
			return
		}

		bus.admin = admin
	}
}

// WithClientOptions 配置认证、TLS 和连接超时，连接地址由 New 的参数指定
func WithClientOptions(options pulsar.ClientOptions) Option {
	return func(bus *Pulbus) {
		bus.clientOptions = options
	}
}

func New(bookie string, opts ...Option) (bus *Pulbus, err error) {
	bus = &Pulbus{
		producers: make(map[string]*producerEntry),
		consumers: make(map[*Consumer]struct{}),
		closing:   make(chan struct{}),
		closeDone: make(chan struct{}),
	}

	// 应用选项
	for _, opt := range opts {
		if opt != nil {
			opt(bus)
		}
	}

	if bus.initErr != nil {
		err = bus.initErr
		bus = nil

		return
	}

	bus.clientOptions.URL = bookie

	bus.client, err = pulsar.NewClient(bus.clientOptions)
	if err != nil {
		bus = nil
	}

	return
}

// Close 关闭所有 producers、consumers 和 client
func (bus *Pulbus) Close() {
	_ = bus.CloseContext(context.Background())
}

// CloseContext 停止创建资源并通知消费退出，ctx 只限制调用方等待资源释放的时间
func (bus *Pulbus) CloseContext(ctx context.Context) (err error) {
	bus.closeOnce.Do(func() {
		bus.lifecycleMu.Lock()
		bus.closed = true
		close(bus.closing)
		bus.lifecycleMu.Unlock()

		go bus.finishClose()
	})

	select {
	case <-bus.closeDone:
		return
	default:
	}

	select {
	case <-bus.closeDone:
	case <-ctx.Done():
		err = ctx.Err()
	}

	return
}

func (bus *Pulbus) finishClose() {
	defer close(bus.closeDone)
	bus.creating.Wait()

	bus.lifecycleMu.Lock()
	producers := bus.producers
	bus.producers = nil
	consumers := bus.consumers
	bus.consumers = nil
	bus.lifecycleMu.Unlock()

	for topic, entry := range producers {
		zap.L().Info("[Pulsar] 关闭 Producer", zap.String("topic", topic))
		entry.producer.Close()
	}

	for consumer := range consumers {
		zap.L().Info("[Pulsar] 关闭 Consumer", zap.String("topic", consumer.key.Topic), zap.String("subscription", consumer.key.Subscription))
		consumer.Close()
	}

	// 关闭 client
	bus.client.Close()
}

// GetAdmin 获取 Pulsar Admin 客户端
func (bus *Pulbus) GetAdmin() *Admin {
	return bus.admin
}
