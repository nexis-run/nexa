// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-28, by liasica

package pulbus

import (
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"go.uber.org/zap"
)

type Pulbus struct {
	client pulsar.Client
	admin  *Admin

	producers sync.Map // map[Topic]pulsar.Producer - 缓存 producer，避免重复创建
	consumers sync.Map // map[ConsumerKey]pulsar.Consumer - 缓存 consumer，避免重复创建
}

// Option Pulbus 配置选项
type Option func(bus *Pulbus)

// WithAdmin 配置 Pulsar Admin 客户端
func WithAdmin(webServiceURL string, opts ...AdminOption) Option {
	return func(bus *Pulbus) {
		admin, err := NewAdmin(webServiceURL, opts...)
		if err != nil {
			zap.L().Error("Pulsar Admin 创建失败", zap.String("webServiceURL", webServiceURL), zap.Error(err))
			return
		}

		bus.admin = admin
	}
}

func New(bookie string, opts ...Option) (bus *Pulbus, err error) {
	var client pulsar.Client

	client, err = pulsar.NewClient(pulsar.ClientOptions{
		URL: bookie,
	})
	if err != nil {
		return
	}

	bus = &Pulbus{
		client:    client,
		producers: sync.Map{},
		consumers: sync.Map{},
	}

	// 应用选项
	for _, opt := range opts {
		opt(bus)
	}

	return
}

// Close 关闭所有 producers、consumers 和 client
func (bus *Pulbus) Close() {
	// 关闭所有 producers
	bus.producers.Range(func(key, value interface{}) bool {
		if producer, ok := value.(pulsar.Producer); ok {
			zap.L().Info("[Pulsar] 关闭 Producer", zap.String("topic", producer.Topic()))
			producer.Close()
		}
		bus.producers.Delete(key)

		return true
	})

	// 关闭所有 consumers
	bus.consumers.Range(func(key, value interface{}) bool {
		if consumer, ok := value.(*Consumer); ok {
			zap.L().Info("[Pulsar] 关闭 Consumer", zap.String("topic", consumer.key.Topic), zap.String("subscription", consumer.key.Subscription))
			consumer.Close()
		}
		bus.consumers.Delete(key)

		return true
	})

	// 关闭 client
	bus.client.Close()
}

// GetAdmin 获取 Pulsar Admin 客户端
func (bus *Pulbus) GetAdmin() *Admin {
	return bus.admin
}
