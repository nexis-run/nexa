package base

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
)

// ResolvePackageName 使用参与当前构建的包声明，空目录按目录名推导
func ResolvePackageName(directory string) (packageName string, err error) {
	var entries []os.DirEntry

	entries, err = os.ReadDir(directory)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		var included bool

		included, err = build.Default.MatchFile(directory, entry.Name())
		if err != nil {
			return
		}

		if !included {
			continue
		}

		fileName := filepath.Join(directory, entry.Name())
		var file *ast.File

		file, err = parser.ParseFile(token.NewFileSet(), fileName, nil, parser.PackageClauseOnly)
		if err != nil {
			err = fmt.Errorf("读取现有包名失败：%w", err)
			return
		}

		if packageName != "" && packageName != file.Name.Name {
			err = fmt.Errorf("目录内存在多个包名：%s", directory)
			return
		}

		packageName = file.Name.Name
	}

	if packageName == "" {
		packageName = strings.ReplaceAll(strings.ToLower(filepath.Base(directory)), "-", "_")
	}

	if !validIdentifier(packageName) {
		err = fmt.Errorf("目录不能推导出有效 Go 包名：%s", directory)
		return
	}

	err = nil

	return
}
