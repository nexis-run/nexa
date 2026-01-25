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

	RootDir        string `json:"-" yaml:"-"` // nexa 项目根目录
	ConfigFileName string `json:"-" yaml:"-"` // 配置文件名称

	OrmClient string `yaml:"ormclient"` // ORM 客户端，默认值：ent.Database

	EntPath     string `yaml:"entPath"`     // ent 目录，默认值：internal/infrastructure/ent
	DaoPath     string `yaml:"daoPath"`     // 数据访问对象目录，默认值：internal/infrastructure/dao
	EchoctxPath string `yaml:"echoctxPath"` // Echo 上下文目录，默认值：internal/app/rest/app

	DI struct {
		Path           string `yaml:"path"`           // 依赖注入生成文件路径，默认值：internal/di/di.go
		DaoProviderSet string `yaml:"daoProviderSet"` // Dao 提供者集合变量名称，默认值：daoProviderSet
	} `yaml:"di"` // 依赖注入配置
}

func defaultConfig() *Config {
	return &Config{
		OrmClient: "ent.Database",

		EntPath:     "internal/infrastructure/ent",
		DaoPath:     "internal/infrastructure/dao",
		EchoctxPath: "internal/app/rest/app",
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
	// 读取配置文件，解析到 Config 结构体
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}

		return nil, fmt.Errorf("配置文件读取失败: %v\n", err)
	}

	var value Config
	err = yaml.Unmarshal(b, &value)
	if err != nil {
		return nil, fmt.Errorf("配置文件解析失败: %v\n", err)
	}

	return &value, nil
}

// InitializeConfig 初始化配置文件，如果配置文件不存在，则使用默认配置
func InitializeConfig(cfgPath string) (err error) {
	var value *Config
	value, err = loadConfig(cfgPath)
	if err != nil {
		return
	}

	cfgOnce.Do(func() {
		cfg = value
		cfg.cfgPath = cfgPath
		cfg.ConfigFileName = filepath.Base(cfgPath)

		cfg.RootDir, _ = filepath.Abs(filepath.Dir(cfgPath))
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

func (c *Config) getAbsPath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}

	return filepath.Abs(filepath.Join(c.RootDir, p))
}

func (c *Config) GetEntPath() (string, error) {
	return c.getAbsPath(c.EntPath)
}

func (c *Config) GetDaoPath() (string, error) {
	return c.getAbsPath(c.DaoPath)
}
