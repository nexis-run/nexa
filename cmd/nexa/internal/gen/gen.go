// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-21, by liasica

package gen

import (
	"os"
	"path/filepath"
	"strings"

	"nexis.run/nexa/cmd/nexa/internal/base"
)

type PackageType string

const (
	PackageDao     PackageType = "dao"
	PackageEchoctx PackageType = "echoctx"
)

// Gen 代码生成器
type Gen struct {
	Config *base.Config
	Module string
}

// New 创建代码生成器
func New() (gen *Gen, err error) {
	gen = &Gen{}

	gen.Config, err = base.GetConfig()
	if err != nil {
		return
	}

	gen.Module = gen.Config.GetModule()

	return
}

// Generate 生成代码
func (gen *Gen) Generate(pt PackageType, name string, force bool, setvars func(g *Gen, c *base.CommonTemplateVariables) any) (err error) {
	var pkgPath string

	switch pt {
	case PackageDao:
		pkgPath = gen.Config.DaoPath
	case PackageEchoctx:
		pkgPath = gen.Config.EchoctxPath
	default:
		return base.ErrUnknownPackageType
	}

	pkg := strings.ReplaceAll(strings.ToLower(filepath.Base(pkgPath)), "-", "_")

	vars := setvars(gen, &base.CommonTemplateVariables{
		Module:  gen.Module,
		Year:    base.GetYear(),
		Date:    base.GetDate(),
		Package: pkg,
	})

	var b []byte

	b, err = RenderTemplate(string(pt)+".tmpl", vars)
	if err != nil {
		return
	}

	filename := filepath.Join(gen.Config.RootDir, pkgPath, strings.ToLower(name+".go"))

	err = base.MkdirAll(filepath.Dir(filename))
	if err != nil {
		return
	}

	_, err = os.Stat(filename)
	if err == nil && !force {
		return base.ErrFileAlreadyExists
	}

	return os.WriteFile(filename, b, 0644)
}
