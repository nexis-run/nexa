package schema

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

type Declaration struct {
	Path     string
	typeSpec *ast.TypeSpec
	aliases  map[string]bool
}

// Names 静态读取嵌入 Ent Schema 或 View 的类型，不执行用户代码
func Names(directory string) (names []string, err error) {
	names = make([]string, 0)
	var declarations map[string]Declaration

	declarations, err = Declarations(directory)
	if err != nil {
		return
	}

	known := make(map[string]bool)

	for changed := true; changed; {
		changed = false

		for name, declaration := range declarations {
			if !known[name] && embedsSchema(declaration, known) {
				known[name] = true
				changed = true
			}
		}
	}

	for name := range known {
		if ast.IsExported(name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)

	return
}

// Declarations 读取参与当前构建的类型声明及其源文件
func Declarations(directory string) (declarations map[string]Declaration, err error) {
	var entries []os.DirEntry

	entries, err = os.ReadDir(directory)
	if os.IsNotExist(err) {
		err = nil
		return
	}

	if err != nil {
		return
	}

	declarations = make(map[string]Declaration)

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

		var file *ast.File
		path := filepath.Join(directory, entry.Name())

		file, err = parser.ParseFile(token.NewFileSet(), path, nil, parser.SkipObjectResolution)
		if err != nil {
			return
		}

		err = addDeclarations(declarations, file, path)
		if err != nil {
			return
		}
	}

	return
}

func addDeclarations(declarations map[string]Declaration, file *ast.File, path string) (err error) {
	aliases := make(map[string]bool)

	for _, spec := range file.Imports {
		var importPath string

		importPath, err = strconv.Unquote(spec.Path.Value)
		if err != nil {
			return
		}

		if importPath != "entgo.io/ent" {
			continue
		}

		alias := "ent"

		if spec.Name != nil {
			alias = spec.Name.Name
		}

		aliases[alias] = true
	}

	for _, declaration := range file.Decls {
		group, isGroup := declaration.(*ast.GenDecl)
		if !isGroup || group.Tok != token.TYPE {
			continue
		}

		for _, spec := range group.Specs {
			typeSpec, isType := spec.(*ast.TypeSpec)
			if !isType {
				continue
			}

			if previous, exists := declarations[typeSpec.Name.Name]; exists {
				err = fmt.Errorf("schema 类型 %s 重复声明：%s 与 %s", typeSpec.Name.Name, previous.Path, path)
				return
			}

			declarations[typeSpec.Name.Name] = Declaration{Path: path, typeSpec: typeSpec, aliases: aliases}
		}
	}

	return
}

func embedsSchema(declaration Declaration, known map[string]bool) bool {
	structType, ok := declaration.typeSpec.Type.(*ast.StructType)
	if !ok {
		return false
	}

	for _, embedded := range structType.Fields.List {
		if len(embedded.Names) != 0 {
			continue
		}

		switch expression := embedded.Type.(type) {
		case *ast.Ident:
			if known[expression.Name] || declaration.aliases["."] && (expression.Name == "Schema" || expression.Name == "View") {
				return true
			}
		case *ast.SelectorExpr:
			identifier, valid := expression.X.(*ast.Ident)
			if valid && declaration.aliases[identifier.Name] && (expression.Sel.Name == "Schema" || expression.Sel.Name == "View") {
				return true
			}
		}
	}

	return false
}
