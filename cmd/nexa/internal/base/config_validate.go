package base

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/mod/module"
)

// Validate 验证配置值、输出位置与模块边界，不要求模块已经初始化
func (c *Config) Validate() (err error) {
	if c == nil || !filepath.IsAbs(c.RootDir) {
		err = errors.New("配置根目录必须是绝对路径")
		return
	}

	if c.OrmClient != "" {
		_, err = parser.ParseExpr(c.OrmClient)
		if err != nil {
			err = fmt.Errorf("ormclient 必须是有效的 Go 表达式：%w", err)
			return
		}
	}

	for _, identifier := range []struct{ name, value string }{
		{"di.daoStructName", c.DI.DaoStructName},
		{"di.daoProviderSetVar", c.DI.DaoProviderSetVar},
	} {
		if !validIdentifier(identifier.value) {
			err = fmt.Errorf("%s 必须是有效的 Go 标识符：%s", identifier.name, identifier.value)
			return
		}
	}

	if c.DI.DaoStructName == c.DI.DaoProviderSetVar {
		err = errors.New("DI 结构体与 provider set 不能使用相同名称")
		return
	}

	var entPath, daoPath, diPath string

	for _, output := range []struct{ name, value string }{
		{"entPath", c.EntPath},
		{"daoPath", c.DaoPath},
		{"echoctxPath", c.EchoctxPath},
	} {
		var absolute string

		absolute, err = c.validateCodePath(output.name, output.value, true)
		if err != nil {
			return
		}

		packageName := filepath.Base(absolute)

		if output.name != "entPath" {
			packageName, err = ResolvePackageName(absolute)
			if err != nil {
				err = fmt.Errorf("%s：%w", output.name, err)
				return
			}
		}

		if !validIdentifier(packageName) || packageName == "main" {
			err = fmt.Errorf("%s 无法生成可导入的 Go 包：%s", output.name, packageName)
			return
		}

		switch output.name {
		case "entPath":
			entPath = absolute
		case "daoPath":
			daoPath = absolute
		}
	}

	diPath, err = c.validateCodePath("di.path", c.DI.Path, false)
	if err != nil {
		return
	}

	filename := filepath.Base(diPath)
	if filepath.Ext(filename) != ".go" || strings.HasSuffix(filename, "_test.go") || strings.HasPrefix(filename, ".") || strings.HasPrefix(filename, "_") {
		err = fmt.Errorf("di.path 必须指向参与构建的 Go 源文件：%s", c.DI.Path)
		return
	}

	if entPath == daoPath || filepath.Dir(diPath) == daoPath || filepath.Dir(diPath) == entPath {
		err = errors.New("Ent、DAO 与 DI 必须使用不同的 Go 包目录")
		return
	}

	err = c.validateEntOptions()

	return
}

func validIdentifier(value string) bool {
	return value != "_" && token.IsIdentifier(value)
}

func validatePathText(name, value string) (err error) {
	if strings.TrimSpace(value) == "" {
		err = fmt.Errorf("%s 路径不能为空", name)
		return
	}

	if strings.ContainsFunc(value, unicode.IsControl) {
		err = fmt.Errorf("%s 路径不能包含控制字符", name)
	}

	return
}

func (c *Config) validateCodePath(name, target string, directory bool) (absolute string, err error) {
	if c == nil || !filepath.IsAbs(c.RootDir) {
		err = errors.New("配置根目录必须是绝对路径")
		return
	}

	err = validatePathText(name, target)
	if err != nil {
		return
	}

	lexicalPath := target
	if !filepath.IsAbs(lexicalPath) {
		lexicalPath = filepath.Join(c.RootDir, lexicalPath)
	}

	var info os.FileInfo

	info, err = os.Lstat(lexicalPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			err = fmt.Errorf("%s 生成目标不能是符号链接：%s", name, lexicalPath)
			return
		}

		if (directory && !info.IsDir()) || (!directory && !info.Mode().IsRegular()) {
			err = fmt.Errorf("%s 路径类型不符合要求：%s", name, lexicalPath)
			return
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		err = fmt.Errorf("%s 路径检查失败：%w", name, err)
		return
	}

	absolute, err = c.GetAbsPath(target)
	if err != nil {
		return
	}

	var root, relative string

	root, err = canonicalPath(c.RootDir)
	if err != nil {
		return
	}

	relative, err = pathWithinRoot(root, absolute)
	if err != nil {
		err = fmt.Errorf("%s：%w", name, err)
		return
	}

	packageDirectory := absolute

	if !directory {
		packageDirectory = filepath.Dir(absolute)
		relative = filepath.Dir(relative)
	}

	err = module.CheckImportPath("example.invalid/" + filepath.ToSlash(relative))
	if relative == "." {
		err = nil
	}

	if err != nil {
		err = fmt.Errorf("%s 不能组成有效的 Go 导入路径：%w", name, err)
		return
	}

	err = rejectNestedModules(root, packageDirectory)

	return
}

func rejectNestedModules(root, directory string) (err error) {
	_, err = pathWithinRoot(root, directory)
	if err != nil {
		return
	}

	for candidate := directory; candidate != root; candidate = filepath.Dir(candidate) {
		moduleFile := filepath.Join(candidate, "go.mod")

		_, err = os.Stat(moduleFile)
		if err == nil {
			err = fmt.Errorf("代码目录位于嵌套的 Go 模块中：%s", moduleFile)
			return
		}

		if !errors.Is(err, os.ErrNotExist) {
			err = fmt.Errorf("嵌套模块检查失败（%s）：%w", moduleFile, err)
			return
		}
	}

	err = nil

	return
}

func (c *Config) validateEntOptions() (err error) {
	for _, feature := range c.EntFeatures {
		if feature == "" || strings.ContainsFunc(feature, unicode.IsSpace) {
			err = fmt.Errorf("特性名称不能包含空白：%q", feature)
			return
		}
	}

	for _, templatePath := range c.EntTemplates {
		var absolute string

		absolute, err = c.GetAbsPath(templatePath)
		if err != nil {
			return
		}

		var info os.FileInfo

		info, err = os.Stat(absolute)
		if err != nil {
			err = fmt.Errorf("模板目录检查失败（%s）：%w", templatePath, err)
			return
		}

		if !info.IsDir() {
			err = fmt.Errorf("模板路径必须是目录：%s", templatePath)
			return
		}
	}

	return
}
