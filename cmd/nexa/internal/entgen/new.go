package entgen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"entgo.io/ent/entc/gen"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
	"nexis.run/nexa/cmd/nexa/internal/schema"
)

// PlanNew 预检所有名称并渲染 schema，不写入项目文件
func (eng *EntGen) PlanNew(names []string, force bool) (files []fileplan.File, err error) {
	if len(names) == 0 {
		err = fmt.Errorf("至少提供一个 schema 名称")
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
		if !base.StringIsExportedIdentifier(name) {
			err = fmt.Errorf("schema 名称必须是大写字母开头的 Go 标识符：%s", name)
			return
		}

		err = gen.ValidSchemaName(name)
		if err != nil {
			return
		}

		key := strings.ToLower(name)
		if seen[key] {
			err = fmt.Errorf("schema 文件名重复：%s", name)
			return
		}

		seen[key] = true
	}

	var entries []os.DirEntry

	entries, err = os.ReadDir(target)
	if err != nil && !os.IsNotExist(err) {
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
		if declaration, exists := declarations[name]; exists && declaration.Path != filepath.Join(target, strings.ToLower(name)+".go") {
			err = fmt.Errorf("schema 类型 %s 已声明在 %s", name, declaration.Path)
			return
		}
	}

	var tmpl *template.Template

	tmpl, err = template.New("schema").Funcs(gen.Funcs).Parse(TemplateNewSchema)
	if err != nil {
		return
	}

	for _, name := range names {
		var buffer bytes.Buffer

		err = tmpl.Execute(&buffer, name)
		if err != nil {
			return
		}

		var content []byte

		content, err = format.Source(buffer.Bytes())
		if err != nil {
			err = fmt.Errorf("格式化 schema %s 失败：%w", name, err)
			return
		}

		files = append(files, fileplan.File{
			Path:      filepath.Join(target, strings.ToLower(name)+".go"),
			Content:   content,
			Overwrite: force,
		})
	}

	var directives []fileplan.File

	directives, err = eng.planGenerateFile(entPath)
	if err == nil {
		files = append(files, directives...)
	}

	return
}
