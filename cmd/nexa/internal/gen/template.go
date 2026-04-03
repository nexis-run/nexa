// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-20, by liasica

package gen

import (
	"bytes"
	"embed"
	"text/template"
)

var (
	//go:embed template/*
	templateFS embed.FS
)

// RenderTemplate 渲染模板
func RenderTemplate(templateName string, data any) (b []byte, err error) {
	var tmpl []byte

	tmpl, err = templateFS.ReadFile("template/" + templateName)
	if err != nil {
		return
	}

	// 创建模板并解析
	var t *template.Template

	t, err = template.New(templateName).Parse(string(tmpl))
	if err != nil {
		return
	}

	// 渲染模板
	var buf bytes.Buffer
	if err = t.Execute(&buf, data); err != nil {
		return
	}

	b = buf.Bytes()

	return
}
