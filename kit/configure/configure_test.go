// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-25, by liasica

package configure

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"nexis.run/nexa/kit"
)

func TestLoad(t *testing.T) {
	type test struct {
		Configure
		Version string
	}

	c, err := Load[test]("../../tests/config.yaml")
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "v1.0.0", c.Version)
	require.Equal(t, "test-app", c.App)
	require.Equal(t, true, c.GetLogger().Stdout)
}

func TestLoggerIsValid(t *testing.T) {
	tests := []struct {
		name     string
		logger   *Logger
		expected bool
	}{
		{name: "nil", logger: nil, expected: false},
		{name: "stdout", logger: &Logger{Stdout: true}, expected: true},
		{name: "disabled kafka", logger: &Logger{Stdout: true, Kafka: &LoggerKafka{Disable: true}}, expected: true},
		{name: "enabled kafka", logger: &Logger{Kafka: &LoggerKafka{Topic: "logs", Brokers: []string{"localhost:9092"}}}, expected: true},
		{name: "incomplete kafka", logger: &Logger{Kafka: &LoggerKafka{Topic: "logs"}}, expected: false},
		{name: "blank topic", logger: &Logger{Kafka: &LoggerKafka{Topic: " ", Brokers: []string{"localhost:9092"}}}, expected: false},
		{name: "blank broker", logger: &Logger{Kafka: &LoggerKafka{Topic: "logs", Brokers: []string{" "}}}, expected: false},
		{name: "no output", logger: &Logger{}, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, test.logger.IsValid())
		})
	}
}

func TestLoadConversionsAndInvalidDocuments(t *testing.T) {
	type config struct {
		Configure
		Timeout time.Duration
		Hosts   []string
		Label   string `koanf:"display_name"`
	}

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "app: example\nenvironment: development\nlogger:\n  stdout: true\ntimeout: 3s\nhosts: one,two\ndisplay_name: label\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	loaded, err := Load[*config](path)
	require.NoError(t, err)
	require.Equal(t, "example", loaded.App)
	require.Equal(t, 3*time.Second, loaded.Timeout)
	require.Equal(t, []string{"one", "two"}, loaded.Hosts)
	require.Equal(t, "label", loaded.Label)

	for _, invalid := range []string{
		"", "null", "[]", "{}",
		content + "---\napp: other", content + "app: duplicate",
		content + "sonyflake_machine_id: 1.5", content + "sonyflake_machine_id: .inf",
	} {
		require.NoError(t, os.WriteFile(path, []byte(invalid), 0600))
		_, err = Load[*config](path)
		require.Error(t, err, invalid)
	}

	require.NoError(t, os.WriteFile(path, []byte("app: ' '\nenvironment: development\nlogger:\n  stdout: true\n"), 0600))
	_, err = Load[config](path)
	require.ErrorIs(t, err, kit.ErrConfigMissName)
}
