// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-27, by liasica

package entgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"nexis.run/nexa/cmd/nexa/internal/base"
)

const (
	genFile = "package ent\n\n//go:generate go run -mod=mod nexis.run/nexa/cmd/nexa@master ent generate\n"
)

type EntGen struct {
	cfg *base.Config
}

func New() (*EntGen, error) {
	cfg, err := base.GetConfig()
	if err != nil {
		return nil, err
	}

	return &EntGen{cfg: cfg}, nil
}

// 创建文件夹
func createDir(target string) error {
	_, err := os.Stat(target)
	if err == nil || !os.IsNotExist(err) {
		return err
	}

	if err = os.MkdirAll(target, os.ModePerm); err != nil {
		return fmt.Errorf("文件夹创建失败: %w", err)
	}

	if err = os.WriteFile(filepath.Join(target, "generate.go"), []byte(genFile), 0644); err != nil {
		return fmt.Errorf("创建 generate.go 文件失败: %w", err)
	}

	return nil
}

// 文件是否存在
func fileExists(target, name string) bool {
	_, err := os.Stat(filepath.Join(target, strings.ToLower(name+".go")))
	return err == nil
}
