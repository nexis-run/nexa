// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-25, by liasica

package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"

	"nexis.run/nexa/cmd/nexa/internal/base"
)

type DaoProvider struct {
	path string

	fset *token.FileSet
	f    *ast.File

	typeName string
	typeSpec *ast.TypeSpec

	variableName string
	variableSpec *ast.ValueSpec
	callExpr     *ast.CallExpr

	Fields []string
}

// ParseDaoProvider 解析 di.go 文件获取 DaoProvider 信息
func ParseDaoProvider(p, typeName, variableName string) (*DaoProvider, error) {
	// 读取文件内容
	content, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}

	// 解析 Go 代码
	fset := token.NewFileSet()
	var f *ast.File
	f, err = parser.ParseFile(fset, p, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	provider := &DaoProvider{
		path:         p,
		fset:         fset,
		f:            f,
		variableName: variableName,
	}

	// 遍历文件中的所有声明
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				// 查找 Dao 结构体
				if d.Tok == token.TYPE {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok && typeSpec.Name.Name == typeName {
						provider.typeSpec = typeSpec
						var structType *ast.StructType
						if structType, ok = typeSpec.Type.(*ast.StructType); ok {
							// 遍历结构体字段
							for _, field := range structType.Fields.List {
								for _, name := range field.Names {
									// 只处理导出的字段（首字母大写）
									if base.StringIsUpperStart(name.Name) {
										provider.Fields = append(provider.Fields, name.Name)
									}
								}
							}
						}
					}
				}

				// 查找变量声明
				if d.Tok == token.VAR {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range valueSpec.Names {
							if name.Name == variableName {
								provider.variableSpec = valueSpec
								if len(valueSpec.Values) > 0 {
									var callExpr *ast.CallExpr
									if callExpr, ok = valueSpec.Values[0].(*ast.CallExpr); ok {
										provider.callExpr = callExpr
									}
								}
							}
						}
					}
				}
			}
		}
	}

	if provider.typeSpec == nil {
		return nil, fmt.Errorf("未找到类型声明: %s", typeName)
	}

	if provider.variableSpec == nil {
		return nil, fmt.Errorf("未找到变量声明: %s", variableName)
	}

	if provider.callExpr == nil {
		return nil, fmt.Errorf("未找到变量调用表达式: %s", variableName)
	}

	return provider, nil
}

// Generate 生成代码（纯 AST 方案 + 自定义格式化）
func (dp *DaoProvider) Generate() ([]byte, error) {
	// 构建新的参数列表
	var newArgs []ast.Expr

	// 添加所有 dao.New{Field} 调用
	for _, field := range dp.Fields {
		newArgs = append(newArgs, &ast.SelectorExpr{
			X:   &ast.Ident{Name: "dao"},
			Sel: &ast.Ident{Name: "New" + field},
		})
	}

	// 添加 wire.Struct(new(Dao), "*")
	wireStructExpr := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "wire"},
			Sel: &ast.Ident{Name: "Struct"},
		},
		Args: []ast.Expr{
			&ast.CallExpr{
				Fun:  &ast.Ident{Name: "new"},
				Args: []ast.Expr{&ast.Ident{Name: "Dao"}},
			},
			&ast.BasicLit{
				Kind:  token.STRING,
				Value: `"*"`,
			},
		},
	}

	newArgs = append(newArgs, wireStructExpr)

	// 替换参数
	dp.callExpr.Args = newArgs

	// 使用自定义格式化器打印整个文件，对 CallExpr 特殊处理
	var buf bytes.Buffer
	formatter := &customFormatter{
		buf:         &buf,
		fset:        dp.fset,
		targetCall:  dp.callExpr,
		fieldsCount: len(dp.Fields),
	}

	if err := formatter.formatFile(dp.f); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// customFormatter 自定义格式化器，对特定的 CallExpr 进行多行格式化
type customFormatter struct {
	buf         *bytes.Buffer
	fset        *token.FileSet
	targetCall  *ast.CallExpr
	fieldsCount int
}

// 格式化代码
func (cf *customFormatter) formatFile(f *ast.File) error {
	// 对于整个文件，我们用 format.Node，但先用自定义逻辑替换目标 CallExpr
	// 创建一个临时缓冲区来生成 wire.NewSet 的字符串表示
	wireCall := cf.buildWireNewSetString()

	// 使用 format.Node 格式化整个文件
	var tempBuf bytes.Buffer
	if err := format.Node(&tempBuf, cf.fset, f); err != nil {
		return fmt.Errorf("格式化 AST 失败: %w", err)
	}

	// 在输出中查找并替换 wire.NewSet 调用
	// 这是一个混合方案：AST 用于结构，字符串替换用于精确控制格式
	output := tempBuf.Bytes()
	output = cf.replaceWireNewSet(output, wireCall)

	cf.buf.Write(output)
	return nil
}

// 创建 wire.NewSet(...) 的多行字符串表示
func (cf *customFormatter) buildWireNewSetString() string {
	var buf bytes.Buffer
	buf.WriteString("wire.NewSet(\n")

	for i, arg := range cf.targetCall.Args {
		// 在最后一个 dao.New 和 wire.Struct 之间加空行
		if i == cf.fieldsCount {
			buf.WriteString("\n")
		}

		buf.WriteString("\t")
		cf.writeExpr(&buf, arg)
		buf.WriteString(",\n")
	}

	buf.WriteString(")")
	return buf.String()
}

// 写入表达式到缓冲区
func (cf *customFormatter) writeExpr(buf *bytes.Buffer, expr ast.Expr) {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		cf.writeExpr(buf, e.X)
		buf.WriteString(".")
		buf.WriteString(e.Sel.Name)
	case *ast.Ident:
		buf.WriteString(e.Name)
	case *ast.CallExpr:
		cf.writeExpr(buf, e.Fun)
		buf.WriteString("(")
		for i, arg := range e.Args {
			if i > 0 {
				buf.WriteString(", ")
			}
			cf.writeExpr(buf, arg)
		}
		buf.WriteString(")")
	case *ast.BasicLit:
		buf.WriteString(e.Value)
	}
}

// 替换 wire.NewSet(...) 部分为自定义格式化的字符串
func (cf *customFormatter) replaceWireNewSet(content []byte, replacement string) []byte {
	// 查找 wire.NewSet( 开始到对应的 ) 结束的部分
	// 使用简单的括号匹配
	wireNewSetPattern := []byte("wire.NewSet(")
	idx := bytes.Index(content, wireNewSetPattern)
	if idx == -1 {
		return content
	}

	// 找到匹配的右括号
	start := idx
	parenCount := 0
	i := idx + len(wireNewSetPattern) - 1
	end := -1

	for i < len(content) {
		if content[i] == '(' {
			parenCount++
		} else if content[i] == ')' {
			parenCount--
			if parenCount == 0 {
				end = i + 1
				break
			}
		}
		i++
	}

	if end == -1 {
		return content
	}

	// 替换
	var result bytes.Buffer
	result.Write(content[:start])
	result.WriteString(replacement)
	result.Write(content[end:])

	return result.Bytes()
}
