// Copyright (C) nexa. 2025-present.
//
// Created at 2025-02-14, by liasica

package internal

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCommand() *cobra.Command {
	var (
		repoUrl string
	)

	c := &cobra.Command{
		Use:     "new",
		Short:   "使用模板创建新项目, 例如: nexa new helloworld",
		GroupID: GroupNew,
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("使用 %s 创建新项目", repoUrl)
		},
	}

	c.Flags().StringVarP(&repoUrl, "repo", "r", "git@gitlab.liasica.com:nexis/nexa-layout.git", "指定模板仓库地址")

	return c
}
