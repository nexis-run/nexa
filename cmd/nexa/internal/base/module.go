// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-19, by liasica

package base

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// GetModule 获取模块信息
func GetModule(dir string) (string, error) {
	// 构造 go.mod 文件路径
	goModPath := filepath.Join(dir, "go.mod")

	// 读取文件内容
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("go.mod 读取失败（%s）：%w", goModPath, err)
	}

	// 解析 go.mod 文件
	var modFile *modfile.File

	modFile, err = modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", fmt.Errorf("go.mod 解析失败（%s）：%w", goModPath, err)
	}

	// 获取模块路径
	if modFile.Module == nil {
		return "", fmt.Errorf("go.mod 中未找到 module 字段信息")
	}

	err = module.CheckImportPath(modFile.Module.Mod.Path)
	if err != nil {
		return "", fmt.Errorf("模块路径无效：%w", err)
	}

	return modFile.Module.Mod.Path, nil
}
