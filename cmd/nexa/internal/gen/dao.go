package gen

import (
	"errors"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
	"nexis.run/nexa/cmd/nexa/internal/parser"
	"nexis.run/nexa/cmd/nexa/internal/schema"
)

type diTemplateVariables struct {
	Package      string
	TypeName     string
	VariableName string
}

func (generator *Gen) PlanDAO(names []string, force bool, withDI bool) (files []fileplan.File, err error) {
	defer func() {
		if err != nil {
			files = nil
		}
	}()

	var directory, packageName, entDirectory, entImport string

	directory, packageName, err = generator.preparePackage(generator.Config.DaoPath, names)
	if err != nil {
		return
	}

	entDirectory, err = generator.Config.GetEntPath()
	if err != nil {
		return
	}

	entImport, err = generator.Config.ResolvePackagePath(entDirectory)
	if err != nil {
		return
	}

	err = validateEntities(entDirectory, names)
	if err != nil {
		return
	}

	for _, name := range names {
		var content []byte

		content, err = renderGo("dao.tmpl", &base.DaoTemplateVariables{
			Package:      packageName,
			EntPkgImport: entImport,
			Name:         name,
			OrmClient:    generator.Config.OrmClient,
		})
		if err != nil {
			return
		}

		files = append(files, fileplan.File{Path: outputPath(directory, name), Content: content, Overwrite: force})
	}

	if withDI {
		var file fileplan.File

		file, err = generator.planDI(names, directory, packageName)
		if err != nil {
			return
		}

		files = append(files, file)
	}

	return
}

func (generator *Gen) planDI(names []string, daoDirectory, daoPackage string) (file fileplan.File, err error) {
	var diPath, daoImport, diPackage string

	diPath, err = generator.Config.GetDIPath()
	if err != nil {
		return
	}

	err = preflightFiles(filepath.Dir(diPath), []string{filepath.Base(diPath)})
	if err != nil {
		return
	}

	_, err = generator.Config.ResolvePackagePath(filepath.Dir(diPath))
	if err != nil {
		return
	}

	daoImport, err = generator.Config.ResolvePackagePath(daoDirectory)
	if err != nil {
		return
	}

	diPackage, err = base.ResolvePackageName(filepath.Dir(diPath))
	if err != nil {
		return
	}

	var content []byte

	content, err = os.ReadFile(diPath)
	overwrite := err == nil

	if errors.Is(err, os.ErrNotExist) {
		content, err = renderGo("di.tmpl", diTemplateVariables{
			Package:      diPackage,
			TypeName:     generator.Config.DI.DaoStructName,
			VariableName: generator.Config.DI.DaoProviderSetVar,
		})
	}

	if err != nil {
		err = fmt.Errorf("准备 DI 文件失败：%w", err)
		return
	}

	var provider *parser.DaoProvider

	provider, err = parser.ParseDaoProvider(parser.DaoProviderConfig{
		Path:         diPath,
		ImportPath:   daoImport,
		PackageName:  daoPackage,
		TypeName:     generator.Config.DI.DaoStructName,
		VariableName: generator.Config.DI.DaoProviderSetVar,
	}, content)
	if err != nil {
		return
	}

	provider.AddField(names...)

	content, err = provider.Generate()
	if err != nil {
		return
	}

	file = fileplan.File{Path: diPath, Content: content, Overwrite: overwrite}

	return
}

func validateEntities(entDirectory string, names []string) error {
	entities := make(map[string]bool)

	schemaNames, err := schema.Names(filepath.Join(entDirectory, "schema"))
	if err != nil {
		return err
	}

	for _, name := range schemaNames {
		entities[name] = true
	}

	var clientContent []byte

	clientContent, err = os.ReadFile(filepath.Join(entDirectory, "client.go"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err == nil {
		var file *ast.File

		file, err = goparser.ParseFile(token.NewFileSet(), "client.go", clientContent, 0)
		if err != nil {
			return fmt.Errorf("解析 Ent client 失败：%w", err)
		}

		collectClientEntities(file, entities)
	}

	for _, name := range names {
		if !entities[name] {
			return fmt.Errorf("未找到 Ent 实体 %s，请先创建 schema 并运行 nexa ent generate", name)
		}
	}

	return nil
}

func collectClientEntities(file *ast.File, entities map[string]bool) {
	ast.Inspect(file, func(node ast.Node) bool {
		specification, ok := node.(*ast.TypeSpec)
		if !ok || specification.Name.Name != "Client" {
			return true
		}

		structure, structureOK := specification.Type.(*ast.StructType)
		if !structureOK {
			return false
		}

		for _, field := range structure.Fields.List {
			star, starOK := field.Type.(*ast.StarExpr)
			if !starOK || len(field.Names) != 1 {
				continue
			}

			identifier, identifierOK := star.X.(*ast.Ident)
			if identifierOK && identifier.Name == field.Names[0].Name+"Client" {
				entities[field.Names[0].Name] = true
			}
		}

		return false
	})
}
