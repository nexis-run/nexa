// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-20, by liasica

package base

import "errors"

var (
	ErrNameRequired        = errors.New("至少需要一个名称")
	ErrInvalidExportedName = errors.New("名称必须是以大写字母开头的 Go 标识符")
)
