// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/apache/pulsar-client-go/pulsaradmin/pkg/utils"
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
	Domain    string // 持久化类型，空值表示 persistent
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
// 例如：persistent://public/default/orders
func (tc TopicConfig) FullName() string {
	domain := tc.Domain
	if domain == "" {
		domain = "persistent"
	}

	return domain + "://" + tc.ShortName()
}

// ShortName 返回短名称（不带 persistent:// 前缀）
// 例如：public/default/orders
func (tc TopicConfig) ShortName() string {
	name := tc.NamespaceFullName() + "/" + tc.Topic
	if tc.Partition >= 0 {
		name += "-partition-" + strconv.Itoa(tc.Partition)
	}

	return name
}

// NamespaceFullName 返回完整的 namespace 路径
// 例如：public/default
func (tc TopicConfig) NamespaceFullName() string {
	return fmt.Sprintf("%s/%s", tc.Tenant, tc.Namespace)
}

// ParseTopic 解析 Topic 字符串，无效输入返回空 Topic
// 需要区分解析错误时使用 ParseTopicName
func ParseTopic(topic string) TopicConfig {
	config, err := ParseTopicName(topic)
	if err != nil {
		return DefaultTopicConfig("")
	}

	return config
}

// ParseTopicName 解析简单名称、namespace/topic 和完整的三段式 Topic 名称
func ParseTopicName(topic string) (config TopicConfig, err error) {
	config = DefaultTopicConfig("")

	topic = strings.TrimSpace(topic)
	if topic == "" {
		err = fmt.Errorf("主题名称不能为空")
		return
	}

	domain := "persistent"

	name := topic
	if prefix, rest, found := strings.Cut(topic, "://"); found {
		domain, name = prefix, rest
	}

	if domain != "persistent" && domain != "non-persistent" {
		err = fmt.Errorf("不支持的 Topic 类型：%s", domain)
		return
	}

	parts := strings.Split(name, "/")
	if strings.Contains(topic, "://") && len(parts) != 3 {
		err = fmt.Errorf("完整 Topic 名称必须包含 tenant/namespace/topic")
		return
	}

	config.Domain = domain

	switch len(parts) {
	case 1:
		config.Topic = parts[0]
	case 2:
		config.Namespace, config.Topic = parts[0], parts[1]
	case 3:
		config.Tenant, config.Namespace, config.Topic = parts[0], parts[1], parts[2]
	default:
		err = fmt.Errorf("主题名称必须是 topic、namespace/topic 或 tenant/namespace/topic")
		return
	}

	_, err = utils.GetNameSpaceName(config.Tenant, config.Namespace)
	if err != nil {
		return
	}

	if index := strings.LastIndex(config.Topic, "-partition-"); index >= 0 {
		suffix := config.Topic[index+len("-partition-"):]

		config.Partition, err = strconv.Atoi(suffix)
		if err != nil || config.Partition < 0 || strconv.Itoa(config.Partition) != suffix {
			err = fmt.Errorf("无效的 Topic 分区号：%s", suffix)
			return
		}

		config.Topic = config.Topic[:index]
	}

	if config.Topic == "" {
		err = fmt.Errorf("主题名称不能为空")
		return
	}

	return
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
	return tb.BuildPartitioned(topic, -1)
}

// BuildPartitioned 构建分区 Topic
func (tb *TopicBuilder) BuildPartitioned(topic string, partition int) string {
	return (TopicConfig{Tenant: tb.tenant, Namespace: tb.namespace, Topic: topic, Partition: partition}).FullName()
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
