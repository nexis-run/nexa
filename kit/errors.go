// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-25, by liasica

package kit

import "errors"

var (
	ErrConfigMissName        = errors.New("需要配置应用名称")
	ErrConfigMissEnvironment = errors.New("需要配置环境变量 <development, staging, production>")
	ErrConfigMissLogger      = errors.New("需要配置日志")
	ErrConfigInvalidLogger   = errors.New("日志配置无效")
	ErrInvalidContext        = errors.New("无效的上下文")
	ErrUnauthorized          = errors.New("未授权")
)
