// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-30, by liasica

package pulbus

import (
	"testing"
)

func TestDefaultTopicConfig(t *testing.T) {
	config := DefaultTopicConfig("orders")

	if config.Tenant != "public" {
		t.Errorf("Expected tenant 'public', got '%s'", config.Tenant)
	}

	if config.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", config.Namespace)
	}

	if config.Topic != "orders" {
		t.Errorf("Expected topic 'orders', got '%s'", config.Topic)
	}

	if config.Partition != -1 {
		t.Errorf("Expected partition -1, got %d", config.Partition)
	}
}

func TestTopicConfigFullName(t *testing.T) {
	tests := []struct {
		name     string
		config   TopicConfig
		expected string
	}{
		{
			name: "non-partitioned topic",
			config: TopicConfig{
				Tenant:    "public",
				Namespace: "default",
				Topic:     "orders",
				Partition: -1,
			},
			expected: "persistent://public/default/orders",
		},
		{
			name: "partitioned topic",
			config: TopicConfig{
				Tenant:    "public",
				Namespace: "default",
				Topic:     "orders",
				Partition: 0,
			},
			expected: "persistent://public/default/orders-partition-0",
		},
		{
			name: "custom namespace",
			config: TopicConfig{
				Tenant:    "production",
				Namespace: "app",
				Topic:     "events",
				Partition: -1,
			},
			expected: "persistent://production/app/events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.FullName()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTopicConfigShortName(t *testing.T) {
	tests := []struct {
		name     string
		config   TopicConfig
		expected string
	}{
		{
			name: "non-partitioned topic",
			config: TopicConfig{
				Tenant:    "public",
				Namespace: "default",
				Topic:     "orders",
				Partition: -1,
			},
			expected: "public/default/orders",
		},
		{
			name: "partitioned topic",
			config: TopicConfig{
				Tenant:    "public",
				Namespace: "default",
				Topic:     "orders",
				Partition: 2,
			},
			expected: "public/default/orders-partition-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ShortName()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTopicConfigNamespaceFullName(t *testing.T) {
	config := TopicConfig{
		Tenant:    "production",
		Namespace: "app",
		Topic:     "events",
		Partition: -1,
	}

	expected := "production/app"
	result := config.NamespaceFullName()

	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestParseTopic(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected TopicConfig
	}{
		{
			name:  "simple topic name",
			input: "orders",
			expected: TopicConfig{
				Tenant:    "public",
				Namespace: "default",
				Topic:     "orders",
				Partition: -1,
			},
		},
		{
			name:  "namespace/topic",
			input: "production/orders",
			expected: TopicConfig{
				Tenant:    "public",
				Namespace: "production",
				Topic:     "orders",
				Partition: -1,
			},
		},
		{
			name:  "tenant/namespace/topic",
			input: "mytenant/mynamespace/mytopic",
			expected: TopicConfig{
				Tenant:    "mytenant",
				Namespace: "mynamespace",
				Topic:     "mytopic",
				Partition: -1,
			},
		},
		{
			name:  "with persistent:// prefix",
			input: "persistent://public/default/events",
			expected: TopicConfig{
				Tenant:    "public",
				Namespace: "default",
				Topic:     "events",
				Partition: -1,
			},
		},
		{
			name:  "with non-persistent:// prefix",
			input: "non-persistent://test/app/notifications",
			expected: TopicConfig{
				Tenant:    "test",
				Namespace: "app",
				Topic:     "notifications",
				Partition: -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTopic(tt.input)

			if result.Tenant != tt.expected.Tenant {
				t.Errorf("Tenant: expected '%s', got '%s'", tt.expected.Tenant, result.Tenant)
			}

			if result.Namespace != tt.expected.Namespace {
				t.Errorf("Namespace: expected '%s', got '%s'", tt.expected.Namespace, result.Namespace)
			}

			if result.Topic != tt.expected.Topic {
				t.Errorf("Topic: expected '%s', got '%s'", tt.expected.Topic, result.Topic)
			}

			if result.Partition != tt.expected.Partition {
				t.Errorf("Partition: expected %d, got %d", tt.expected.Partition, result.Partition)
			}
		})
	}
}

func TestTopicBuilder(t *testing.T) {
	builder := NewTopicBuilder("production", "app")

	t.Run("Build", func(t *testing.T) {
		expected := "persistent://production/app/orders"

		result := builder.Build("orders")
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("BuildPartitioned", func(t *testing.T) {
		expected := "persistent://production/app/orders-partition-3"

		result := builder.BuildPartitioned("orders", 3)
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})

	t.Run("Namespace", func(t *testing.T) {
		expected := "production/app"

		result := builder.Namespace()
		if result != expected {
			t.Errorf("Expected '%s', got '%s'", expected, result)
		}
	})
}

func TestNamespaceConfig(t *testing.T) {
	ns := GetNamespace("production", "app")

	if ns.Tenant != "production" {
		t.Errorf("Expected tenant 'production', got '%s'", ns.Tenant)
	}

	if ns.Namespace != "app" {
		t.Errorf("Expected namespace 'app', got '%s'", ns.Namespace)
	}

	expected := "production/app"

	result := ns.FullName()
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestNamespaceConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"DefaultNamespace", DefaultNamespace, "public/default"},
		{"ProductionNamespace", ProductionNamespace, "production/app"},
		{"DevelopmentNamespace", DevelopmentNamespace, "development/app"},
		{"TestNamespace", TestNamespace, "test/app"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("Expected %s to be '%s', got '%s'", tt.name, tt.expected, tt.value)
			}
		})
	}
}
