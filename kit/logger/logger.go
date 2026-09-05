// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package logger

import (
	"context"
	"errors"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"nexis.run/nexa/kit"
	"nexis.run/nexa/kit/configure"
)

var (
	setupMu           sync.Mutex
	activeKafkaWriter *KafkaWriter
	managedWriters    = make(map[*KafkaWriter]struct{})
)

func Setup(cfg *configure.Logger) (err error) {
	if !cfg.IsValid() {
		err = kit.ErrConfigInvalidLogger
		return
	}

	var cores []zapcore.Core
	var kafkaWriter *KafkaWriter

	if cfg.Stdout {
		consoleCore := zapcore.NewCore(
			ConsoleEncoder(),
			zapcore.Lock(os.Stdout),
			zapcore.DebugLevel,
		)
		cores = append(cores, consoleCore)
	}

	if cfg.Kafka != nil && !cfg.Kafka.Disable {
		kafkaEncoderConfig := zap.NewProductionEncoderConfig()

		kafkaEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		kafkaEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

		kafkaEncoder := zapcore.NewJSONEncoder(kafkaEncoderConfig)
		kafkaWriter = NewKafkaWriter(cfg.Kafka.Brokers, cfg.Kafka.Topic)

		kafkaCore := zapcore.NewCore(
			kafkaEncoder,
			zapcore.AddSync(kafkaWriter),
			zapcore.InfoLevel,
		)

		cores = append(cores, kafkaCore)
	}

	l := zap.New(zapcore.NewTee(cores...), zap.AddCaller())

	if cfg.Name != "" {
		l = l.Named(cfg.Name)
	}

	// 替换全局日志器，旧 writer 独立完成关闭
	setupMu.Lock()
	previous := activeKafkaWriter
	activeKafkaWriter = kafkaWriter

	if kafkaWriter != nil {
		managedWriters[kafkaWriter] = struct{}{}
	}

	zap.ReplaceGlobals(l)
	setupMu.Unlock()

	if previous != nil {
		previous.startClose()
	}

	return
}

// Close 停止所有在管 Kafka writer，等待当前及配置替换前的日志完成发送尝试
func Close(ctx context.Context) (err error) {
	setupMu.Lock()
	writers := make([]*KafkaWriter, 0, len(managedWriters)+1)

	for writer := range managedWriters {
		writers = append(writers, writer)
	}

	if _, managed := managedWriters[activeKafkaWriter]; activeKafkaWriter != nil && !managed {
		writers = append(writers, activeKafkaWriter)
	}
	setupMu.Unlock()

	for _, writer := range writers {
		writer.startClose()
	}

	for _, writer := range writers {
		err = errors.Join(err, writer.waitClose(ctx))
	}

	return
}
