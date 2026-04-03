// Copyright (C) godoc. 2026-present.
//
// Created at 2026-01-12, by liasica

package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/nexa/internal/base"
)

func ConfigCmd() (*cobra.Group, *cobra.Command) {
	g := &cobra.Group{
		ID:    "config",
		Title: "配置管理命令",
	}

	cmd := &cobra.Command{
		Use:               "config",
		Short:             "管理配置",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		GroupID:           g.ID,
	}

	cmd.AddCommand(configInitCmd())

	return g, cmd
}

func configInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "init",
		Short:             "初始化配置",
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		Run: func(_ *cobra.Command, _ []string) {
			cfg, _ := base.GetConfig()
			p := cfg.GetConfigFilePath()

			// 检测配置文件是否已存在
			if _, err := os.Stat(p); err == nil {
				fmt.Println("配置文件已存在")
				os.Exit(1)
			}

			// 写入默认配置
			defaultCfg := base.DefaultConfig()
			fmt.Printf("默认配置:\n%s\n", defaultCfg)

			err := os.WriteFile(p, []byte(defaultCfg), os.ModePerm)
			if err != nil {
				fmt.Printf("写入配置文件失败: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("配置文件创建成功")
		},
	}
}
