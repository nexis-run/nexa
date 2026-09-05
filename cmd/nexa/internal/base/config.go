// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-19, by liasica

package base

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const DefaultConfigFile = ".nexa.yaml"

type LoadOptions struct {
	ConfigPath   string
	Explicit     bool
	AllowMissing bool
}

type Config struct {
	cfgPath string

	RootDir        string `json:"-" yaml:"-"` // Go 模块根目录，未初始化模块时为配置文件目录
	ConfigFileName string `json:"-" yaml:"-"` // 配置文件名称

	OrmClient string `json:"ormclient" yaml:"ormclient"` // 空值表示通过构造参数注入 Ent 客户端

	EntPath      string   `json:"entPath" yaml:"entPath"`
	EntTemplates []string `json:"entTemplates" yaml:"entTemplates"`
	EntFeatures  []string `json:"entFeatures" yaml:"entFeatures"`
	DaoPath      string   `json:"daoPath" yaml:"daoPath"`
	EchoctxPath  string   `json:"echoctxPath" yaml:"echoctxPath"`
	DI           DI       `json:"di" yaml:"di"`
}

type DI struct {
	Path              string `json:"path" yaml:"path"`
	DaoProviderSetVar string `json:"daoProviderSetVar" yaml:"daoProviderSetVar"`
	DaoStructName     string `json:"daoStructName" yaml:"daoStructName"`
}

func defaultConfig() *Config {
	return &Config{
		EntPath:      "internal/infrastructure/ent",
		EntTemplates: []string{},
		EntFeatures:  []string{},
		DaoPath:      "internal/infrastructure/dao",
		EchoctxPath:  "internal/app/rest/app",
		DI: DI{
			Path:              "internal/di/di.go",
			DaoProviderSetVar: "daoProviderSet",
			DaoStructName:     "Dao",
		},
	}
}

// NewDefaultConfig 创建指定目标的默认配置，不读取目标文件内容
func NewDefaultConfig(path string) (cfg *Config, err error) {
	if path == "" {
		path = DefaultConfigFile
	}

	var absolutePath, root string

	absolutePath, err = absoluteConfigPath(path)
	if err != nil {
		return
	}

	root, err = findModuleRoot(filepath.Dir(absolutePath))
	if err != nil {
		return
	}

	root, err = canonicalPath(root)
	if err != nil {
		return
	}

	cfg = defaultConfig()
	cfg.cfgPath = absolutePath
	cfg.RootDir = root
	cfg.ConfigFileName = filepath.Base(absolutePath)

	return
}

// LoadConfig 加载独立配置，不修改全局配置或工作目录
func LoadConfig(opts LoadOptions) (cfg *Config, err error) {
	var path string

	path, err = discoverConfigPath(opts)
	if err != nil {
		return
	}

	cfg, err = NewDefaultConfig(path)
	if err != nil {
		return
	}

	var info os.FileInfo

	info, err = os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && (opts.AllowMissing || isImplicitDefault(opts)) {
			_, err = os.Lstat(path)
			if errors.Is(err, os.ErrNotExist) {
				err = cfg.Validate()
				return
			}

			if err == nil {
				err = fmt.Errorf("配置文件的符号链接目标不存在：%s", path)
			}

			return
		}

		err = fmt.Errorf("配置文件读取失败（%s）：%w", path, err)

		return
	}

	if !info.Mode().IsRegular() {
		err = fmt.Errorf("配置路径不是普通文件：%s", path)
		return
	}

	var content []byte

	content, err = os.ReadFile(path)
	if err != nil {
		err = fmt.Errorf("配置文件读取失败（%s）：%w", path, err)
		return
	}

	err = decodeConfig(content, cfg)
	if err != nil {
		err = fmt.Errorf("配置文件解析失败（%s）：%w", path, err)
		return
	}

	err = cfg.Validate()

	return
}

func decodeConfig(content []byte, cfg *Config) (err error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var document yaml.Node

	err = decoder.Decode(&document)
	if errors.Is(err, io.EOF) {
		err = errors.New("配置文件不能为空")
		return
	}

	if err != nil {
		return
	}

	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		err = errors.New("配置根节点必须是 YAML 映射")
		return
	}

	err = rejectNullValues(&document, make(map[*yaml.Node]bool))
	if err != nil {
		return
	}

	var extra yaml.Node

	err = decoder.Decode(&extra)
	if err == nil {
		err = errors.New("配置文件只能包含一个 YAML 文档")
		return
	}

	if !errors.Is(err, io.EOF) {
		return
	}

	decoder = yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	err = decoder.Decode(cfg)

	return
}

func rejectNullValues(node *yaml.Node, visited map[*yaml.Node]bool) (err error) {
	if node == nil || visited[node] {
		return
	}

	visited[node] = true
	if node.Tag == "!!null" {
		err = fmt.Errorf("第 %d 行的配置值不能为 null", node.Line)
		return
	}

	if node.Alias != nil {
		err = rejectNullValues(node.Alias, visited)
		return
	}

	for _, child := range node.Content {
		err = rejectNullValues(child, visited)
		if err != nil {
			return
		}
	}

	return
}

// MarshalYAMLBytes 仅序列化配置字段
func (c *Config) MarshalYAMLBytes() ([]byte, error) {
	if c == nil {
		return nil, errors.New("配置不能为空")
	}

	return yaml.Marshal(c)
}

// ConfigExists 报告配置目标是否为现有普通文件
func (c *Config) ConfigExists() bool {
	if c == nil {
		return false
	}

	info, err := os.Stat(c.cfgPath)

	return err == nil && info.Mode().IsRegular()
}

func (c *Config) GetConfigFilePath() string {
	return c.cfgPath
}

// ResolveModule 验证配置与文件系统并读取 Go 模块路径
func (c *Config) ResolveModule() (module string, err error) {
	err = c.Validate()
	if err != nil {
		return
	}

	module, err = GetModule(c.RootDir)

	return
}
