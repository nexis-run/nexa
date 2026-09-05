// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-20, by liasica

package base

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"io"
	"strings"
)

func StringIsExportedIdentifier(name string) bool {
	return token.IsIdentifier(name) && ast.IsExported(name)
}

// ValidateNames 预检批量名称、文件名冲突与隐式构建约束
func ValidateNames(names []string) error {
	if len(names) == 0 {
		return ErrNameRequired
	}

	for index, name := range names {
		if !StringIsExportedIdentifier(name) {
			return fmt.Errorf("%w：%s", ErrInvalidExportedName, name)
		}

		if !portableSourceName(strings.ToLower(name) + ".go") {
			return fmt.Errorf("名称 %s 会生成测试文件或平台专属文件，请使用不含特殊后缀的名称", name)
		}

		for _, previous := range names[:index] {
			if strings.EqualFold(previous, name) {
				return fmt.Errorf("名称重复或生成文件名冲突：%s、%s", previous, name)
			}
		}
	}

	return nil
}

func portableSourceName(filename string) bool {
	if strings.HasSuffix(filename, "_test.go") {
		return false
	}

	// 两个平台的系统与架构均不同，带任一隐式平台约束的文件至少被一个平台排除
	for _, target := range []struct{ system, arch string }{{"linux", "amd64"}, {"windows", "arm64"}} {
		buildContext := build.Context{
			GOOS:   target.system,
			GOARCH: target.arch,
			OpenFile: func(string) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader("package generated\n")), nil
			},
		}

		included, err := buildContext.MatchFile("", filename)
		if err != nil || !included {
			return false
		}
	}

	return true
}
