package gen

import (
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"nexis.run/nexa/cmd/nexa/internal/fileplan"
)

// preflightDeclarations 检查整批输出与包内其他文件的顶层声明
func preflightDeclarations(directory string, files []fileplan.File) error {
	declarations := make(map[string]string)
	targets := make(map[string]bool)

	for _, file := range files {
		targets[file.Path] = true

		err := collectDeclarations(declarations, file.Path, file.Content)
		if err != nil {
			return err
		}
	}

	entries, err := os.ReadDir(directory)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return err
	}

	for _, entry := range entries {
		filename := filepath.Join(directory, entry.Name())
		if entry.IsDir() || targets[filename] || !strings.HasSuffix(filename, ".go") || strings.HasSuffix(filename, "_test.go") {
			continue
		}

		var included bool

		included, err = build.Default.MatchFile(directory, entry.Name())
		if err != nil {
			return err
		}

		if !included {
			continue
		}

		err = collectDeclarations(declarations, filename, nil)
		if err != nil {
			return err
		}
	}

	return nil
}

func collectDeclarations(declarations map[string]string, filename string, content []byte) error {
	var source any

	if content != nil {
		source = content
	}

	file, err := parser.ParseFile(token.NewFileSet(), filename, source, parser.SkipObjectResolution)
	if err != nil {
		return err
	}

	for _, declaration := range file.Decls {
		var names []*ast.Ident

		switch node := declaration.(type) {
		case *ast.FuncDecl:
			if node.Recv == nil && node.Name.Name != "init" {
				names = append(names, node.Name)
			}
		case *ast.GenDecl:
			for _, spec := range node.Specs {
				switch specification := spec.(type) {
				case *ast.TypeSpec:
					names = append(names, specification.Name)
				case *ast.ValueSpec:
					names = append(names, specification.Names...)
				}
			}
		}

		for _, name := range names {
			if name.Name == "_" {
				continue
			}

			if previous, exists := declarations[name.Name]; exists {
				return fmt.Errorf("声明 %s 冲突：%s 与 %s", name.Name, previous, filename)
			}

			declarations[name.Name] = filename
		}
	}

	return nil
}
