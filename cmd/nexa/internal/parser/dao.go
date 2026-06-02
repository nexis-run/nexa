// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-25, by liasica

package parser

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"regexp"
	"slices"
	"strings"
)

type DaoProvider struct {
	path         string
	importPath   string
	typeName     string
	variableName string
	fields       []string
	content      []byte
}

// NewDaoProvider 解析 di.go 文件获取 DaoProvider 信息
func NewDaoProvider(diPath, typeName, variableName, importPath string) (*DaoProvider, error) {
	// 读取文件内容
	content, err := os.ReadFile(diPath)
	if err != nil {
		return nil, err
	}

	if typeName == "" || variableName == "" || importPath == "" {
		return nil, fmt.Errorf("typeName / variableName / importPath 不能为空")
	}

	provider := &DaoProvider{
		path:         diPath,
		importPath:   importPath,
		typeName:     typeName,
		variableName: variableName,
		content:      content,
	}

	// 使用正则提取现有的 Dao 结构体字段
	structPattern := regexp.MustCompile(`(?s)type\s+` + typeName + `\s+struct\s*\{([^}]*)\}`)

	matches := structPattern.FindSubmatch(content)
	if len(matches) > 1 {
		// 提取字段
		fieldPattern := regexp.MustCompile(`(\w+)\s+\*dao\.(\w+)Dao`)

		fieldMatches := fieldPattern.FindAllSubmatch(matches[1], -1)
		for _, fm := range fieldMatches {
			if len(fm) > 1 {
				fieldName := string(fm[1])
				// 只处理导出的字段（首字母大写）
				if len(fieldName) > 0 && fieldName[0] >= 'A' && fieldName[0] <= 'Z' {
					provider.fields = append(provider.fields, fieldName)
				}
			}
		}
	}

	return provider, nil
}

// AddField 添加字段
func (dp *DaoProvider) AddField(fields ...string) {
	dp.fields = append(dp.fields, fields...)
	// 去重
	fieldSet := make(map[string]struct{})

	uniqueFields := make([]string, 0, len(dp.fields))
	for _, f := range dp.fields {
		if _, exists := fieldSet[f]; !exists {
			fieldSet[f] = struct{}{}
			uniqueFields = append(uniqueFields, f)
		}
	}

	dp.fields = uniqueFields

	// 排序
	slices.Sort(dp.fields)
}

// Generate 生成代码
func (dp *DaoProvider) Generate() ([]byte, error) {
	content := string(dp.content)

	// 检查并添加 import
	if dp.importPath != "" && !dp.hasImport(content) {
		content = dp.addImport(content)
	}

	// 替换 type Dao struct 定义
	content = dp.replaceStruct(content)

	// 替换 wire.NewSet 调用
	content = dp.replaceProviderSet(content)

	// 使用 gofmt 格式化最终结果
	formatted, err := format.Source([]byte(content))
	if err != nil {
		return nil, fmt.Errorf("格式化失败: %w", err)
	}

	return formatted, nil
}

// WriteToFile 将生成的代码写回文件
func (dp *DaoProvider) WriteToFile() error {
	b, err := dp.Generate()
	if err != nil {
		return err
	}

	return os.WriteFile(dp.path, b, 0644)
}

// hasImport 检查是否已经引入了 dao 包
func (dp *DaoProvider) hasImport(content string) bool {
	// 匹配 import 中是否有目标路径
	pattern := regexp.MustCompile(`(?s)import\s*\((.*?)\)`)

	matches := pattern.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.Contains(matches[1], dp.importPath)
	}

	// 单行 import
	singlePattern := regexp.MustCompile(`import\s+"` + regexp.QuoteMeta(dp.importPath) + `"`)

	return singlePattern.MatchString(content)
}

// addImport 添加 import 语句
func (dp *DaoProvider) addImport(content string) string {
	// 查找 import 块
	importPattern := regexp.MustCompile(`(?s)(import\s*\(\s*)(.*?)(\s*\))`)
	if importPattern.MatchString(content) {
		// 已有 import 块，在最后添加
		return importPattern.ReplaceAllStringFunc(content, func(match string) string {
			parts := importPattern.FindStringSubmatch(match)
			if len(parts) > 3 {
				imports := strings.TrimSpace(parts[2])
				if imports != "" {
					return parts[1] + imports + "\n\t\"" + dp.importPath + "\"\n" + parts[3]
				}

				return parts[1] + "\"" + dp.importPath + "\"\n" + parts[3]
			}

			return match
		})
	}

	// 没有 import 块，在 package 后添加
	packagePattern := regexp.MustCompile(`(package\s+\w+)`)

	return packagePattern.ReplaceAllString(content, "$1\n\nimport (\n\t\""+dp.importPath+"\"\n)")
}

// replaceStruct 替换 Dao 结构体定义
func (dp *DaoProvider) replaceStruct(content string) string {
	newStruct := dp.buildDaoStruct()

	// 使用智能括号匹配来处理结构体定义
	// 首先找到 type Xxx struct { 的位置
	typePattern := regexp.MustCompile(`type\s+` + regexp.QuoteMeta(dp.typeName) + `\s+struct\s*\{`)

	loc := typePattern.FindStringIndex(content)
	if loc == nil {
		return content
	}

	// 从左大括号开始，找到匹配的右大括号
	start := loc[1] - 1 // 左大括号的位置
	depth := 1
	end := start + 1

	for end < len(content) && depth > 0 {
		switch content[end] {
		case '{':
			depth++
		case '}':
			depth--
		}

		end++
	}

	if depth != 0 {
		// 没有找到匹配的大括号，返回原内容
		return content
	}

	// 替换整个 type Xxx struct {...} 部分
	return content[:loc[0]] + "type " + newStruct + content[end:]
}

// replaceProviderSet 替换 wire.NewSet 调用
func (dp *DaoProvider) replaceProviderSet(content string) string {
	newCall := dp.buildProviderSetCall()

	// 使用更智能的方法匹配嵌套括号
	// 首先找到 var xxx = wire.NewSet( 的位置
	varPattern := regexp.MustCompile(`var\s+` + regexp.QuoteMeta(dp.variableName) + `\s*=\s*wire\.NewSet\(`)

	loc := varPattern.FindStringIndex(content)
	if loc == nil {
		return content
	}

	// 从左括号开始，找到匹配的右括号
	start := loc[1] - 1 // 左括号的位置
	depth := 1
	end := start + 1

	for end < len(content) && depth > 0 {
		switch content[end] {
		case '(':
			depth++
		case ')':
			depth--
		}

		end++
	}

	if depth != 0 {
		// 没有找到匹配的括号，返回原内容
		return content
	}

	// 替换整个 var xxx = wire.NewSet(...) 部分
	return content[:loc[0]] + "var " + dp.variableName + " = " + newCall + content[end:]
}

// buildDaoStruct 构建 Dao 结构体定义
func (dp *DaoProvider) buildDaoStruct() string {
	var buf bytes.Buffer
	_, _ = fmt.Fprintf(&buf, "%s struct {\n", dp.typeName)

	for _, field := range dp.fields {
		_, _ = fmt.Fprintf(&buf, "\t%s *dao.%sDao\n", field, field)
	}
	buf.WriteString("}")

	return buf.String()
}

// buildProviderSetCall 创建参数列表
func (dp *DaoProvider) buildProviderSetCall() string {
	var buf bytes.Buffer
	buf.WriteString("wire.NewSet(\n")

	for _, field := range dp.fields {
		_, _ = fmt.Fprintf(&buf, "\tdao.New%s,\n", field)
	}

	_, _ = fmt.Fprintf(&buf, "\n\twire.Struct(new(%s), \"*\"),\n)", dp.typeName)

	return buf.String()
}
