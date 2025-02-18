// Copyright (C) nexa. 2025-present.
//
// Created at 2025-02-17, by liasica

package internal

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/cli/internal/gitlab"
)

func UpgradeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:     "upgrade",
		Short:   "升级, 例如: nexa upgrade [ mod ]",
		GroupID: "upgrade",
		Run: func(_ *cobra.Command, _ []string) {
		},
	}

	c.AddCommand(upgradeModCommand)

	return c
}

var upgradeModCommand = &cobra.Command{
	Use:   "mod",
	Short: "升级已有项目nexa依赖, 例如: nexa upgrade mod",
	Run: func(_ *cobra.Command, _ []string) {
		id, err := gitlab.New(os.Getenv("GITLAB_ACCESS_TOKEN"))
		if err != nil {
			fmt.Printf("创建Gitlab客户端失败: %v\n", err)
			os.Exit(1)
		}

		
	},
}
