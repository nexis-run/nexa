// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-25, by liasica

package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

type DaoProvider struct {
	path         string
	importPath   string
	typeName     string
	variableName string
	daoPackage   string
	wirePackage  string
	fields       []string
	content      []byte
}

type DaoProviderConfig struct {
	Path         string
	ImportPath   string
	PackageName  string
	TypeName     string
	VariableName string
}

// NewDaoProvider 解析 di.go 文件获取 DaoProvider 信息
func NewDaoProvider(diPath, typeName, variableName, importPath string) (provider *DaoProvider, err error) {
	var content []byte

	content, err = os.ReadFile(diPath)
	if err != nil {
		return
	}

	provider, err = ParseDaoProvider(DaoProviderConfig{
		Path:         diPath,
		ImportPath:   importPath,
		PackageName:  strings.ReplaceAll(strings.ToLower(path.Base(importPath)), "-", "_"),
		TypeName:     typeName,
		VariableName: variableName,
	}, content)

	return
}

// ParseDaoProvider 从已有内容创建依赖注入更新器
func ParseDaoProvider(config DaoProviderConfig, content []byte) (provider *DaoProvider, err error) {
	for _, name := range []string{config.TypeName, config.VariableName, config.PackageName} {
		if !token.IsIdentifier(name) || name == "_" {
			err = fmt.Errorf("依赖注入标识符无效：%s", name)
			return
		}
	}

	if config.ImportPath == "" {
		err = fmt.Errorf("DAO 导入路径不能为空")
		return
	}

	provider = &DaoProvider{
		path:         config.Path,
		importPath:   config.ImportPath,
		typeName:     config.TypeName,
		variableName: config.VariableName,
		content:      bytes.Clone(content),
	}

	var file *ast.File

	_, file, err = provider.parseFile()
	if err != nil {
		return
	}

	structType := findStruct(file, config.TypeName)
	if structType == nil {
		err = fmt.Errorf("未找到结构体 %s", config.TypeName)
		return
	}

	provider.daoPackage = availableImportName(file, config.PackageName)
	if hasImport(file, config.ImportPath) {
		provider.daoPackage = importName(file, config.ImportPath, config.PackageName)
	}

	provider.wirePackage = importName(file, "github.com/google/wire", "wire")

	if provider.daoPackage == "." || provider.daoPackage == "_" {
		err = fmt.Errorf("DAO 包必须通过包名导入：%s", config.ImportPath)
		return
	}

	if provider.wirePackage == "." || provider.wirePackage == "_" || !hasImport(file, "github.com/google/wire") {
		err = fmt.Errorf("依赖注入文件必须通过包名导入 github.com/google/wire")
		return
	}

	if findProviderSet(file, config.VariableName, provider.wirePackage) == nil {
		err = fmt.Errorf("未找到依赖注入变量 %s", config.VariableName)
		return
	}

	return
}

// AddField 添加字段
func (dp *DaoProvider) AddField(fields ...string) {
	dp.fields = append(dp.fields, fields...)
	fieldSet := make(map[string]struct{})
	uniqueFields := make([]string, 0, len(dp.fields))

	for _, field := range dp.fields {
		if _, exists := fieldSet[field]; exists {
			continue
		}

		fieldSet[field] = struct{}{}
		uniqueFields = append(uniqueFields, field)
	}

	slices.Sort(uniqueFields)
	dp.fields = uniqueFields
}

// Generate 生成代码
func (dp *DaoProvider) Generate() (result []byte, err error) {
	var (
		file *ast.File
		fset *token.FileSet
	)

	fset, file, err = dp.parseFile()
	if err != nil {
		return
	}

	structType := findStruct(file, dp.typeName)
	if structType == nil {
		err = fmt.Errorf("未找到结构体 %s", dp.typeName)
		return
	}

	providerSet := findProviderSet(file, dp.variableName, dp.wirePackage)
	if providerSet == nil {
		err = fmt.Errorf("未找到依赖注入变量 %s", dp.variableName)
		return
	}

	var addedFields []string

	addedFields, err = dp.addStructFields(structType)
	if err != nil {
		return
	}

	if len(addedFields) == 0 {
		result = dp.content
		return
	}

	if !hasImport(file, dp.importPath) {
		astutil.AddNamedImport(fset, file, dp.daoPackage, dp.importPath)
	}

	dp.addProviderSetEntries(providerSet, addedFields)

	var buffer bytes.Buffer

	err = format.Node(&buffer, fset, file)
	if err != nil {
		err = fmt.Errorf("格式化失败：%w", err)
		return
	}

	result = buffer.Bytes()

	return
}

func (dp *DaoProvider) parseFile() (fset *token.FileSet, file *ast.File, err error) {
	fset = token.NewFileSet()

	file, err = goparser.ParseFile(fset, dp.path, dp.content, goparser.ParseComments)
	if err != nil {
		err = fmt.Errorf("解析依赖注入文件失败：%w", err)
	}

	return
}

func (dp *DaoProvider) addStructFields(structType *ast.StructType) (addedFields []string, err error) {
	existingFields := make(map[string]*ast.Field)
	position := structType.Fields.Closing - 1

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			existingFields[embeddedFieldName(field.Type)] = field
		}

		for _, name := range field.Names {
			existingFields[name.Name] = field
		}
	}

	for _, fieldName := range dp.fields {
		if !token.IsIdentifier(fieldName) || !ast.IsExported(fieldName) {
			err = fmt.Errorf("DAO 字段名称无效：%s", fieldName)
			return
		}

		if field, exists := existingFields[fieldName]; exists {
			if len(field.Names) == 0 || !isDaoField(field, fieldName, dp.daoPackage) {
				err = fmt.Errorf("结构体字段 %s 已存在且类型不是 *%s.%sDao", fieldName, dp.daoPackage, fieldName)
				return
			}

			continue
		}

		structType.Fields.List = append(structType.Fields.List, &ast.Field{
			Names: []*ast.Ident{newIdentifier(fieldName, position)},
			Type: &ast.StarExpr{
				Star: position,
				X: &ast.SelectorExpr{
					X:   newIdentifier(dp.daoPackage, position),
					Sel: newIdentifier(fieldName+"Dao", position),
				},
			},
		})
		addedFields = append(addedFields, fieldName)
	}

	return
}

func (dp *DaoProvider) addProviderSetEntries(providerSet *ast.CallExpr, addedFields []string) {
	existingProviders := make(map[string]struct{})
	position := providerSet.Rparen - 1

	for _, argument := range providerSet.Args {
		if fieldName, ok := daoProviderFieldName(argument, dp.daoPackage); ok {
			existingProviders[fieldName] = struct{}{}
		}
	}

	for _, fieldName := range addedFields {
		if _, exists := existingProviders[fieldName]; exists {
			continue
		}

		providerSet.Args = append(providerSet.Args, &ast.SelectorExpr{
			X:   newIdentifier(dp.daoPackage, position),
			Sel: newIdentifier("New"+fieldName, position),
		})
	}

	structProvider := findStructProvider(providerSet, dp.typeName, dp.wirePackage)
	if structProvider == nil {
		providerSet.Args = append(providerSet.Args, newStructProvider(dp.typeName, position, dp.wirePackage, addedFields))
		return
	}

	if structProviderIncludesAll(structProvider) {
		return
	}

	existingFields := structProviderFields(structProvider)

	for _, fieldName := range addedFields {
		if _, exists := existingFields[fieldName]; exists {
			continue
		}

		structProvider.Args = append(structProvider.Args, &ast.BasicLit{
			ValuePos: structProvider.Rparen - 1,
			Kind:     token.STRING,
			Value:    strconv.Quote(fieldName),
		})
	}
}

func findStruct(file *ast.File, typeName string) (structType *ast.StructType) {
	for _, declaration := range file.Decls {
		genericDeclaration, ok := declaration.(*ast.GenDecl)
		if !ok || genericDeclaration.Tok != token.TYPE {
			continue
		}

		for _, specification := range genericDeclaration.Specs {
			typeSpecification, specificationOK := specification.(*ast.TypeSpec)
			if !specificationOK || typeSpecification.Name.Name != typeName {
				continue
			}

			structType, _ = typeSpecification.Type.(*ast.StructType)

			return
		}
	}

	return
}

func findProviderSet(file *ast.File, variableName, wirePackage string) (providerSet *ast.CallExpr) {
	for _, declaration := range file.Decls {
		genericDeclaration, ok := declaration.(*ast.GenDecl)
		if !ok || genericDeclaration.Tok != token.VAR {
			continue
		}

		for _, specification := range genericDeclaration.Specs {
			valueSpecification, specificationOK := specification.(*ast.ValueSpec)
			if !specificationOK {
				continue
			}

			for index, name := range valueSpecification.Names {
				if name.Name != variableName || index >= len(valueSpecification.Values) {
					continue
				}

				call, callOK := valueSpecification.Values[index].(*ast.CallExpr)
				if callOK && isSelector(call.Fun, wirePackage, "NewSet") {
					return call
				}
			}
		}
	}

	return
}

func findStructProvider(providerSet *ast.CallExpr, typeName, wirePackage string) (structProvider *ast.CallExpr) {
	for _, argument := range providerSet.Args {
		call, callOK := argument.(*ast.CallExpr)
		if !callOK || !isSelector(call.Fun, wirePackage, "Struct") || len(call.Args) == 0 {
			continue
		}

		if address, ok := call.Args[0].(*ast.UnaryExpr); ok && address.Op == token.AND {
			if literal, literalOK := address.X.(*ast.CompositeLit); literalOK {
				if identifier, identifierOK := literal.Type.(*ast.Ident); identifierOK && identifier.Name == typeName {
					return call
				}
			}
		}

		newCall, newCallOK := call.Args[0].(*ast.CallExpr)
		if !newCallOK || len(newCall.Args) != 1 {
			continue
		}

		identifier, identifierOK := newCall.Fun.(*ast.Ident)
		if !identifierOK || identifier.Name != "new" {
			continue
		}

		structIdentifier, structIdentifierOK := newCall.Args[0].(*ast.Ident)
		if structIdentifierOK && structIdentifier.Name == typeName {
			return call
		}
	}

	return
}

func newStructProvider(typeName string, position token.Pos, wirePackage string, fields []string) *ast.CallExpr {
	provider := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   newIdentifier(wirePackage, position),
			Sel: newIdentifier("Struct", position),
		},
		Lparen: position,
		Args: []ast.Expr{
			&ast.CallExpr{
				Fun:    newIdentifier("new", position),
				Lparen: position,
				Args:   []ast.Expr{newIdentifier(typeName, position)},
				Rparen: position,
			},
		},
		Rparen: position,
	}

	for _, field := range fields {
		provider.Args = append(provider.Args, &ast.BasicLit{ValuePos: position, Kind: token.STRING, Value: strconv.Quote(field)})
	}

	return provider
}

func newIdentifier(name string, position token.Pos) *ast.Ident {
	return &ast.Ident{NamePos: position, Name: name}
}

func isDaoField(field *ast.Field, fieldName, daoPackage string) bool {
	starExpression, starExpressionOK := field.Type.(*ast.StarExpr)
	if !starExpressionOK {
		return false
	}

	selector, selectorOK := starExpression.X.(*ast.SelectorExpr)
	if !selectorOK || selector.Sel.Name != fieldName+"Dao" {
		return false
	}

	packageIdentifier, packageIdentifierOK := selector.X.(*ast.Ident)

	return packageIdentifierOK && packageIdentifier.Name == daoPackage
}

func daoProviderFieldName(expression ast.Expr, daoPackage string) (fieldName string, ok bool) {
	selector, selectorOK := expression.(*ast.SelectorExpr)
	if !selectorOK {
		return
	}

	packageIdentifier, packageIdentifierOK := selector.X.(*ast.Ident)
	if !packageIdentifierOK || packageIdentifier.Name != daoPackage || len(selector.Sel.Name) <= len("New") {
		return
	}

	if selector.Sel.Name[:len("New")] != "New" {
		return
	}

	fieldName = selector.Sel.Name[len("New"):]
	ok = true

	return
}

func isSelector(expression ast.Expr, packageName, functionName string) bool {
	selector, selectorOK := expression.(*ast.SelectorExpr)
	if !selectorOK || selector.Sel.Name != functionName {
		return false
	}

	packageIdentifier, packageIdentifierOK := selector.X.(*ast.Ident)

	return packageIdentifierOK && packageIdentifier.Name == packageName
}

func structProviderIncludesAll(structProvider *ast.CallExpr) bool {
	for _, argument := range structProvider.Args[1:] {
		literal, ok := argument.(*ast.BasicLit)
		if ok && literal.Kind == token.STRING {
			value, err := strconv.Unquote(literal.Value)
			if err == nil && value == "*" {
				return true
			}
		}
	}

	return false
}

func structProviderFields(structProvider *ast.CallExpr) map[string]struct{} {
	fields := make(map[string]struct{})

	for _, argument := range structProvider.Args[1:] {
		literal, ok := argument.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			continue
		}

		fieldName, err := strconv.Unquote(literal.Value)
		if err == nil {
			fields[fieldName] = struct{}{}
		}
	}

	return fields
}

func importName(file *ast.File, importPath, fallback string) string {
	for _, importSpecification := range file.Imports {
		importedPath, valid := unquoteImportPath(importSpecification)
		if !valid || importedPath != importPath {
			continue
		}

		if importSpecification.Name != nil {
			return importSpecification.Name.Name
		}

		if fallback != "" {
			return fallback
		}

		return strings.ReplaceAll(strings.ToLower(path.Base(importPath)), "-", "_")
	}

	return fallback
}

func hasImport(file *ast.File, importPath string) bool {
	for _, specification := range file.Imports {
		if importedPath, valid := unquoteImportPath(specification); valid && importedPath == importPath {
			return true
		}
	}

	return false
}

func availableImportName(file *ast.File, preferred string) string {
	names := make(map[string]bool)

	for _, specification := range file.Imports {
		if importedPath, valid := unquoteImportPath(specification); valid {
			names[importName(file, importedPath, "")] = true
		}
	}

	for suffix := 0; ; suffix++ {
		candidate := preferred

		if suffix > 0 {
			candidate += strconv.Itoa(suffix)
		}

		if !names[candidate] && file.Scope.Lookup(candidate) == nil {
			return candidate
		}
	}
}

func embeddedFieldName(expression ast.Expr) string {
	switch field := expression.(type) {
	case *ast.Ident:
		return field.Name
	case *ast.StarExpr:
		return embeddedFieldName(field.X)
	case *ast.SelectorExpr:
		return field.Sel.Name
	default:
		return ""
	}
}

func unquoteImportPath(importSpecification *ast.ImportSpec) (path string, valid bool) {
	var err error

	path, err = strconv.Unquote(importSpecification.Path.Value)
	valid = err == nil

	return
}
