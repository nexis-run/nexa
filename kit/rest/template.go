// Copyright (C) nexa. 2025-present.
//
// Created at 2025-08-04, by liasica

package rest

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"strings"

	"github.com/labstack/echo/v4"
)

type HTMLTemplate struct {
	Templates map[string]*template.Template
}

func (t *HTMLTemplate) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return t.Templates[name].ExecuteTemplate(w, name, data)
}

// LoadTemplates 从嵌入的文件系统中加载HTML模板
// 使用例子:
//
// e.Renderer = rest.LoadTemplates(assets.TemplateFS, "templates")
//
//	e.GET("/docs/openapi.yaml", func(c echo.Context) error {
//			return c.String(http.StatusOK, assets.OpenApiFile)
//	})
func LoadTemplates(tmpls embed.FS, templatesDir string) (ht *HTMLTemplate) {
	ht = &HTMLTemplate{Templates: make(map[string]*template.Template)}

	_ = fs.WalkDir(tmpls, templatesDir, func(path string, d fs.DirEntry, _ error) (err error) {
		if d.IsDir() {
			return
		}

		name := strings.Replace(path, templatesDir+"/", "", 1)
		pt := template.New(name)
		b, _ := tmpls.ReadFile(path)

		_, _ = pt.Parse(string(b))

		ht.Templates[name] = pt

		return
	})

	return
}
