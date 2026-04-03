// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-17, by liasica

package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func ConsoleEncoder() zapcore.Encoder {
	config := zap.NewDevelopmentEncoderConfig()

	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		s, ok := levelToCapitalColorString[l]
		if !ok {
			s = unknownLevel(l)
		}
		enc.AppendString(s)
	}

	return zapcore.NewConsoleEncoder(config)
}
