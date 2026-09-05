// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-20, by liasica

package gen

import (
	"bytes"
	"embed"
	"sync"
	"text/template"
)

var (
	//go:embed template/*
	templateFS      embed.FS
	parsedTemplates = sync.OnceValues(func() (*template.Template, error) {
		return template.New("nexa").Option("missingkey=error").ParseFS(templateFS, "template/*.tmpl")
	})
)

// RenderTemplate 渲染模板
func RenderTemplate(templateName string, data any) (b []byte, err error) {
	var t *template.Template

	t, err = parsedTemplates()
	if err != nil {
		return
	}

	// 渲染模板
	var buf bytes.Buffer
	if err = t.ExecuteTemplate(&buf, templateName, data); err != nil {
		return
	}

	b = buf.Bytes()

	return
}
