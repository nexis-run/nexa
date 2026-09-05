package base

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/mod/module"
)

func isImplicitDefault(opts LoadOptions) bool {
	return !opts.Explicit && (opts.ConfigPath == "" || opts.ConfigPath == DefaultConfigFile)
}

func discoverConfigPath(opts LoadOptions) (resolved string, err error) {
	if opts.Explicit && opts.ConfigPath == "" {
		err = errors.New("显式配置路径不能为空")
		return
	}

	target := opts.ConfigPath
	if target == "" {
		target = DefaultConfigFile
	}

	resolved, err = absoluteConfigPath(target)
	if err != nil || !isImplicitDefault(opts) {
		return
	}

	var root string

	root, err = findModuleRoot(filepath.Dir(resolved))
	if err != nil {
		return
	}

	for directory := filepath.Dir(resolved); ; directory = filepath.Dir(directory) {
		resolved = filepath.Join(directory, DefaultConfigFile)
		var info os.FileInfo

		info, err = os.Stat(resolved)
		if err == nil {
			if !info.Mode().IsRegular() {
				err = fmt.Errorf("配置路径不是普通文件：%s", resolved)
				return
			}

			return
		}

		if !errors.Is(err, os.ErrNotExist) {
			err = fmt.Errorf("配置路径检查失败（%s）：%w", resolved, err)
			return
		}

		if directory == root {
			err = nil
			return
		}
	}
}

func absoluteConfigPath(target string) (configPath string, err error) {
	err = validatePathText("配置文件", target)
	if err != nil {
		return
	}

	configPath, err = filepath.Abs(target)
	if err != nil {
		return
	}

	var parent string

	parent, err = canonicalPath(filepath.Dir(configPath))
	if err != nil {
		return
	}

	configPath = filepath.Join(parent, filepath.Base(configPath))

	return
}

func findModuleRoot(directory string) (root string, err error) {
	root = directory

	for candidate := directory; ; candidate = filepath.Dir(candidate) {
		var info os.FileInfo
		moduleFile := filepath.Join(candidate, "go.mod")

		info, err = os.Stat(moduleFile)
		if err == nil {
			if !info.Mode().IsRegular() {
				err = fmt.Errorf("模块文件不是普通文件：%s", moduleFile)
				return
			}

			root = candidate

			return
		}

		if !errors.Is(err, os.ErrNotExist) {
			err = fmt.Errorf("模块文件检查失败（%s）：%w", moduleFile, err)
			return
		}

		if filepath.Dir(candidate) == candidate {
			err = nil
			return
		}
	}
}

// canonicalPath 解析现存祖先的物理路径，并保留尚未创建的后缀
func canonicalPath(target string) (resolved string, err error) {
	if !filepath.IsAbs(target) {
		err = fmt.Errorf("路径必须是绝对路径：%s", target)
		return
	}

	var missing []string
	candidate := filepath.Clean(target)

	for {
		_, err = os.Lstat(candidate)
		if err == nil {
			resolved, err = filepath.EvalSymlinks(candidate)
			if err != nil {
				err = fmt.Errorf("符号链接解析失败（%s）：%w", candidate, err)
				return
			}

			break
		}

		if !errors.Is(err, os.ErrNotExist) || filepath.Dir(candidate) == candidate {
			err = fmt.Errorf("路径检查失败（%s）：%w", candidate, err)
			return
		}

		missing = append(missing, filepath.Base(candidate))
		candidate = filepath.Dir(candidate)
	}

	for index := len(missing) - 1; index >= 0; index-- {
		resolved = filepath.Join(resolved, missing[index])
	}

	return
}

func (c *Config) GetAbsPath(target string) (resolved string, err error) {
	if c == nil || !filepath.IsAbs(c.RootDir) {
		err = errors.New("配置根目录必须是绝对路径")
		return
	}

	err = validatePathText("配置路径", target)
	if err != nil {
		return
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(c.RootDir, target)
	}

	resolved, err = canonicalPath(target)

	return
}

func (c *Config) GetEntPath() (string, error) {
	return c.GetAbsPath(c.EntPath)
}

func (c *Config) GetDaoPath() (string, error) {
	return c.GetAbsPath(c.DaoPath)
}

func (c *Config) GetDIPath() (string, error) {
	return c.GetAbsPath(c.DI.Path)
}

// ResolvePackagePath 返回经过模块边界检查的导入路径
func (c *Config) ResolvePackagePath(target string) (packagePath string, err error) {
	if c == nil || !filepath.IsAbs(c.RootDir) {
		err = errors.New("配置根目录必须是绝对路径")
		return
	}

	var modulePath string

	modulePath, err = GetModule(c.RootDir)
	if err != nil {
		return
	}

	var root, absolute, relative string

	root, err = canonicalPath(c.RootDir)
	if err != nil {
		return
	}

	absolute, err = c.validateCodePath("代码目录", target, true)
	if err != nil {
		return
	}

	relative, err = filepath.Rel(root, absolute)
	if err != nil {
		return
	}

	packagePath = path.Join(modulePath, filepath.ToSlash(relative))

	err = module.CheckImportPath(packagePath)
	if err != nil {
		err = fmt.Errorf("导入路径无效（%s）：%w", packagePath, err)
	}

	return
}

func pathWithinRoot(root, target string) (relative string, err error) {
	relative, err = filepath.Rel(root, target)
	if err != nil {
		return
	}

	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		err = fmt.Errorf("代码路径必须位于 Go 模块目录内：%s", target)
	}

	return
}
