// Copyright (C) micros. 2025-present.
//
// Created at 2025-02-10, by liasica

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/cli/internal"
)

func main() {
	cmd := &cobra.Command{
		Use:               "nexa",
		CompletionOptions: cobra.CompletionOptions{
			// DisableDefaultCmd: true,
		},
	}

	cmd.AddCommand(
		internal.NewCommand(),
		internal.UpgradeCommand(),
	)

	cmd.AddGroup(
		&cobra.Group{
			ID:    internal.GroupNew,
			Title: "新建",
		},
		&cobra.Group{
			ID:    internal.GroupUpgrade,
			Title: "升级",
		},
	)

	err := cmd.Execute()
	if err != nil {
		fmt.Printf("命令执行失败: %v\n", err)
		os.Exit(1)
	}
}
