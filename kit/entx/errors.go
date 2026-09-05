// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-27, by liasica

package entx

import "errors"

var (
	ErrHardDeleteForbidden        = errors.New("禁止硬删除")
	ErrSoftDeleteQueryUnsupported = errors.New("查询生成代码不支持软删除过滤")
)
