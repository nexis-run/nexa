// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-19, by liasica

package base

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

// ParseModFile 读取并解析目录下的 go.mod
func ParseModFile(dir string) (modFile *modfile.File, err error) {
	goModPath := filepath.Join(dir, "go.mod")

	var data []byte

	data, err = os.ReadFile(goModPath)
	if err != nil {
		err = fmt.Errorf("go.mod 读取失败（%s）：%w", goModPath, err)
		return
	}

	modFile, err = modfile.Parse(goModPath, data, nil)
	if err != nil {
		err = fmt.Errorf("go.mod 解析失败（%s）：%w", goModPath, err)
	}

	return
}

// GetModule 读取并校验 go.mod 声明的模块路径
func GetModule(dir string) (modulePath string, err error) {
	var modFile *modfile.File

	modFile, err = ParseModFile(dir)
	if err != nil {
		return
	}

	if modFile.Module == nil {
		err = errors.New("go.mod 中未找到 module 字段信息")
		return
	}

	err = module.CheckImportPath(modFile.Module.Mod.Path)
	if err != nil {
		err = fmt.Errorf("模块路径无效：%w", err)
		return
	}

	modulePath = modFile.Module.Mod.Path

	return
}
