// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-25, by liasica

package authz

import "errors"

var (
	ErrUnauthorized   = errors.New("未授权用户")
	ErrForbidden      = errors.New("无权限访问")
	ErrNotInitialized = errors.New("权限客户端未初始化")
	ErrEmptyResponse  = errors.New("权限服务返回空响应")
)
