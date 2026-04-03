// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-27, by liasica

package entgen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"entgo.io/ent/entc/gen"
)

func (eng *EntGen) New(names []string) (err error) {
	var target string

	target, err = eng.cfg.GetEntPath()
	if err != nil {
		return
	}

	target = filepath.Join(target, "schema")

	tmpl := template.New("schema").Funcs(gen.Funcs)

	tmpl, err = tmpl.Parse(TemplateNewSchema)
	if err != nil {
		return
	}

	err = createDir(target)
	if err != nil {
		return
	}

	for _, name := range names {
		if err = gen.ValidSchemaName(name); err != nil {
			return fmt.Errorf("创建 %s 失败: %w", name, err)
		}

		if fileExists(target, name) {
			return fmt.Errorf("创建 %s 失败: 已存在", name)
		}

		b := bytes.NewBuffer(nil)
		if err = tmpl.Execute(b, name); err != nil {
			return fmt.Errorf("模板执行失败: %w", err)
		}

		newFileTarget := filepath.Join(target, strings.ToLower(name+".go"))
		if err = os.WriteFile(newFileTarget, b.Bytes(), 0644); err != nil {
			return fmt.Errorf("文件写入失败: %w", err)
		}

		fmt.Printf("创建 %s 成功", name)
	}

	return
}
