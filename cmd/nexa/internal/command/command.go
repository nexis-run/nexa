// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-20, by liasica

package command

import (
	"strings"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/nexa/internal/base"
)

// 代码示例格式化
func examples(ex ...string) string {
	for i := range ex {
		ex[i] = "  " + ex[i] // indent each row with 2 spaces.
	}

	return strings.Join(ex, "\n")
}

// 检查参数是否以大写字母开头
func isUpperStartArgs(_ *cobra.Command, args []string) error {
	for _, name := range args {
		if !base.StringIsUpperStart(name) {
			return base.ErrNameMustStartWithUpper
		}
	}

	return nil
}
