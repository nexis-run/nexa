// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"fmt"
	"strings"
)

// 常用的 Namespace 配置
const (
	// DefaultNamespace 默认 namespace
	DefaultNamespace = "public/default"

	// ProductionNamespace 生产环境 namespace
	ProductionNamespace = "production/app"

	// DevelopmentNamespace 开发环境 namespace
	DevelopmentNamespace = "development/app"

	// TestNamespace 测试环境 namespace
	TestNamespace = "test/app"
)

// TopicConfig Topic 配置
type TopicConfig struct {
	Tenant    string // 租户，默认 "public"
	Namespace string // 命名空间，默认 "default"
	Topic     string // Topic 名称
	Partition int    // 分区号，-1 表示非分区 Topic
}

// DefaultTopicConfig 返回默认的 Topic 配置
func DefaultTopicConfig(topic string) TopicConfig {
	return TopicConfig{
		Tenant:    "public",
		Namespace: "default",
		Topic:     topic,
		Partition: -1,
	}
}

// FullName 返回完整的 Topic 路径
// 例如: persistent://public/default/orders
func (tc TopicConfig) FullName() string {
	if tc.Partition >= 0 {
		return fmt.Sprintf("persistent://%s/%s/%s-partition-%d",
			tc.Tenant, tc.Namespace, tc.Topic, tc.Partition)
	}

	return fmt.Sprintf("persistent://%s/%s/%s",
		tc.Tenant, tc.Namespace, tc.Topic)
}

// ShortName 返回短名称（不带 persistent:// 前缀）
// 例如: public/default/orders
func (tc TopicConfig) ShortName() string {
	if tc.Partition >= 0 {
		return fmt.Sprintf("%s/%s/%s-partition-%d",
			tc.Tenant, tc.Namespace, tc.Topic, tc.Partition)
	}

	return fmt.Sprintf("%s/%s/%s",
		tc.Tenant, tc.Namespace, tc.Topic)
}

// NamespaceFullName 返回完整的 namespace 路径
// 例如: public/default
func (tc TopicConfig) NamespaceFullName() string {
	return fmt.Sprintf("%s/%s", tc.Tenant, tc.Namespace)
}

// ParseTopic 解析 Topic 字符串
// 支持以下格式:
//   - "orders" -> public/default/orders
//   - "tenant/namespace/topic" -> tenant/namespace/topic
//   - "persistent://tenant/namespace/topic" -> tenant/namespace/topic
func ParseTopic(topic string) TopicConfig {
	config := TopicConfig{
		Tenant:    "public",
		Namespace: "default",
		Partition: -1,
	}

	// 去除 persistent:// 前缀
	topic = strings.TrimPrefix(topic, "persistent://")
	topic = strings.TrimPrefix(topic, "non-persistent://")

	parts := strings.Split(topic, "/")

	switch len(parts) {
	case 1:
		// 只有 topic 名称
		config.Topic = parts[0]
	case 2:
		// namespace/topic
		config.Namespace = parts[0]
		config.Topic = parts[1]
	case 3:
		// tenant/namespace/topic
		config.Tenant = parts[0]
		config.Namespace = parts[1]
		config.Topic = parts[2]
	}

	return config
}

// TopicBuilder Topic 构建器
type TopicBuilder struct {
	tenant    string
	namespace string
}

// NewTopicBuilder 创建 Topic 构建器
func NewTopicBuilder(tenant, namespace string) *TopicBuilder {
	return &TopicBuilder{
		tenant:    tenant,
		namespace: namespace,
	}
}

// Build 构建 Topic 完整路径
func (tb *TopicBuilder) Build(topic string) string {
	return fmt.Sprintf("persistent://%s/%s/%s", tb.tenant, tb.namespace, topic)
}

// BuildPartitioned 构建分区 Topic
func (tb *TopicBuilder) BuildPartitioned(topic string, partition int) string {
	return fmt.Sprintf("persistent://%s/%s/%s-partition-%d",
		tb.tenant, tb.namespace, topic, partition)
}

// Namespace 返回 namespace 完整路径
func (tb *TopicBuilder) Namespace() string {
	return fmt.Sprintf("%s/%s", tb.tenant, tb.namespace)
}

// NamespaceConfig Namespace 配置
type NamespaceConfig struct {
	Tenant    string
	Namespace string
}

// FullName 返回完整的 namespace 路径
func (nc NamespaceConfig) FullName() string {
	return fmt.Sprintf("%s/%s", nc.Tenant, nc.Namespace)
}

// GetNamespace 获取 Namespace 配置
func GetNamespace(tenant, namespace string) NamespaceConfig {
	return NamespaceConfig{
		Tenant:    tenant,
		Namespace: namespace,
	}
}
