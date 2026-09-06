package entgen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"entgo.io/ent/entc/gen"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
	"nexis.run/nexa/cmd/nexa/internal/schema"
)

var schemaTemplate = sync.OnceValues(func() (*template.Template, error) {
	return template.New("schema").Funcs(gen.Funcs).Parse(TemplateNewSchema)
})

// schemaTemplateVariables 定义 schema 模板变量
type schemaTemplateVariables struct {
	Name       string
	SoftDelete bool
}

// PlanNew 预检所有名称并渲染 schema，不写入项目文件
func (eng *EntGen) PlanNew(names []string, force, softDelete bool) (files []fileplan.File, err error) {
	err = base.ValidateNames(names)
	if err != nil {
		return
	}

	var entPath string

	entPath, err = eng.cfg.GetEntPath()
	if err != nil {
		return
	}

	target := filepath.Join(entPath, "schema")
	seen := make(map[string]bool)

	for _, name := range names {
		err = gen.ValidSchemaName(name)
		if err != nil {
			return
		}

		key := strings.ToLower(name)
		seen[key] = true
	}

	var entries []os.DirEntry

	entries, err = os.ReadDir(target)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}

	for _, entry := range entries {
		filename := strings.ToLower(entry.Name())
		if !strings.HasSuffix(filename, ".go") || !seen[strings.TrimSuffix(filename, ".go")] {
			continue
		}

		if filename != entry.Name() {
			err = fmt.Errorf("schema 文件名大小写冲突：%s", filepath.Join(target, entry.Name()))
			return
		}
	}

	var declarations map[string]schema.Declaration

	declarations, err = schema.Declarations(target)
	if err != nil {
		return
	}

	for _, name := range names {
		if declaration, exists := declarations[name]; exists && declaration.Path != schemaFilePath(target, name) {
			err = fmt.Errorf("schema 类型 %s 已声明在 %s", name, declaration.Path)
			return
		}
	}

	var tmpl *template.Template

	tmpl, err = schemaTemplate()
	if err != nil {
		return
	}

	for _, name := range names {
		var buffer bytes.Buffer

		err = tmpl.Execute(&buffer, schemaTemplateVariables{Name: name, SoftDelete: softDelete})
		if err != nil {
			return
		}

		var content []byte

		content, err = format.Source(buffer.Bytes())
		if err != nil {
			err = fmt.Errorf("格式化 schema %s 失败：%w", name, err)
			return
		}

		files = append(files, fileplan.File{Path: schemaFilePath(target, name), Content: content, Overwrite: force})
	}

	var directives []fileplan.File

	directives, err = eng.planGenerateFile(entPath)
	if err == nil {
		files = append(files, directives...)
	}

	return
}

func schemaFilePath(directory, name string) string {
	return filepath.Join(directory, strings.ToLower(name)+".go")
}
