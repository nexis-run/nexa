// Copyright (C) nexa. 2025-present.
//
// Created at 2025-08-04, by liasica

package rest

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type HTMLTemplate struct {
	Templates map[string]*template.Template
}

func (t *HTMLTemplate) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	if t == nil || t.Templates[name] == nil {
		return fmt.Errorf("HTML 模板不存在：%s", name)
	}

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

	_ = fs.WalkDir(tmpls, templatesDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			zap.L().Error("遍历模板目录失败", zap.String("path", path), zap.Error(walkErr))
			return nil
		}

		if d.IsDir() {
			return nil
		}

		name := strings.Replace(path, templatesDir+"/", "", 1)

		b, err := tmpls.ReadFile(path)
		if err != nil {
			zap.L().Error("读取模板文件失败", zap.String("path", path), zap.Error(err))
			return nil
		}

		var pt *template.Template

		pt, err = template.New(name).Parse(string(b))
		if err != nil {
			zap.L().Error("解析模板失败", zap.String("name", name), zap.Error(err))
			return nil
		}

		ht.Templates[name] = pt

		return nil
	})

	return
}
