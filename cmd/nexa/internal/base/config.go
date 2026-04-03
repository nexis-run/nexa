// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-19, by liasica

package base

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	cfg     *Config
	cfgOnce sync.Once
)

type Config struct {
	cfgPath string
	module  string

	RootDir        string `json:"-" yaml:"-"` // nexa 项目根目录
	ConfigFileName string `json:"-" yaml:"-"` // 配置文件名称

	OrmClient string `yaml:"ormclient"` // ORM 客户端，默认值：ent.Database

	EntPath      string   `yaml:"entPath"`      // ent 目录，默认值：internal/infrastructure/ent
	EntTemplates []string `yaml:"entTemplates"` // ent 代码生成模板目录
	EntFeatures  []string `yaml:"entFeatures"`  // ent 代码生成特性

	DaoPath     string `yaml:"daoPath"`     // 数据访问对象目录，默认值：internal/infrastructure/dao
	EchoctxPath string `yaml:"echoctxPath"` // Echo 上下文目录，默认值：internal/app/rest/app

	DI DI `yaml:"di"` // 依赖注入配置
}

type DI struct {
	Path              string `yaml:"path"`              // 依赖注入生成文件路径，默认值：internal/di/di.go
	DaoProviderSetVar string `yaml:"daoProviderSetVar"` // Dao 提供者集合变量名称，默认值：daoProviderSet
	DaoStructName     string `yaml:"daoStructName"`     // Dao 结构体名称，默认值：Dao
}

func defaultConfig() *Config {
	return &Config{
		OrmClient: "ent.Database",

		EntPath:     "internal/infrastructure/ent",
		DaoPath:     "internal/infrastructure/dao",
		EchoctxPath: "internal/app/rest/app",

		DI: DI{
			Path:              "internal/di/di.go",
			DaoProviderSetVar: "daoProviderSet",
			DaoStructName:     "Dao",
		},
	}
}

// DefaultConfig 返回默认配置的 YAML 字符串
func DefaultConfig() string {
	defaultCfg := defaultConfig()
	b, _ := yaml.Marshal(defaultCfg)

	return string(b)
}

// 读取 YAML 配置文件。如果路径为空，则使用当前工作目录下的 ".nexa.yaml"。
func loadConfig(cfgPath string) (*Config, error) {
	c := defaultConfig()

	// 读取配置文件，解析到 Config 结构体
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}

		return nil, fmt.Errorf("配置文件读取失败: %v", err)
	}

	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, fmt.Errorf("配置文件解析失败: %v", err)
	}

	return c, nil
}

// InitializeConfig 初始化配置文件，如果配置文件不存在，则使用默认配置
func InitializeConfig(cfgPath string) (err error) {
	var value *Config

	value, err = loadConfig(cfgPath)
	if err != nil {
		return
	}

	value.cfgPath = cfgPath
	value.ConfigFileName = filepath.Base(cfgPath)

	value.RootDir, err = filepath.Abs(filepath.Dir(cfgPath))
	if err != nil {
		return
	}

	value.module, err = GetModule(filepath.Dir(cfgPath))
	if err != nil {
		return
	}

	cfgOnce.Do(func() {
		cfg = value
	})

	return
}

// GetConfig 获取全局配置实例
func GetConfig() (*Config, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置未初始化，请先调用 InitializeConfig")
	}

	return cfg, nil
}

// GetConfigFilePath 获取配置文件路径
func (c *Config) GetConfigFilePath() string {
	return c.cfgPath
}

func (c *Config) GetAbsPath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}

	return filepath.Abs(filepath.Join(c.RootDir, p))
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

func (c *Config) GetModule() string {
	return c.module
}

func (c *Config) GetPkgPath(p string) string {
	return GetPkgPath(c.module, c.RootDir, p)
}

func (c *Config) GetEntPkgPath() string {
	return c.GetPkgPath(c.EntPath)
}

func (c *Config) GetDaoPkgPath() string {
	return c.GetPkgPath(c.DaoPath)
}
