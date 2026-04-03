// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-25, by liasica

package configure

import (
	"os"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/sony/sonyflake/v2"

	"nexis.run/nexa/kit"
)

func init() {
	// 设置全局时区
	tz := "Asia/Shanghai"
	_ = os.Setenv("TZ", tz)
	loc, _ := time.LoadLocation(tz)
	time.Local = loc
}

type Configure struct {
	App         string          // 应用名称
	Environment kit.Environment // 环境变量
	Logger      *Logger         // 日志配置
}

type Configurable interface {
	GetApp() string
	GetEnvironment() kit.Environment
	GetLogger() *Logger
}

func (c Configure) GetApp() string {
	return c.App
}

func (c Configure) GetEnvironment() kit.Environment {
	return c.Environment
}

func (c Configure) GetLogger() *Logger {
	return c.Logger
}

type Logger struct {
	Name string // 日志名称

	Stdout bool // 是否输出到控制台

	// 输出至kafka
	Kafka *LoggerKafka
}

type LoggerKafka struct {
	Disable bool     // 是否禁用kafka日志输出
	Topic   string   // kafka topic
	Brokers []string // kafka brokers
}

func (l *Logger) IsVaild() (vaild bool) {
	if l == nil {
		return
	}

	// 如果没有配置 stdout 和 kafka / kafka未启用，则无效
	if !l.Stdout && (l.Kafka == nil || l.Kafka.Disable) {
		return
	}

	// 如果配置了kafka，topic和name不能为空
	if l.Kafka != nil {
		return l.Kafka.Topic != "" && len(l.Kafka.Brokers) > 0
	}

	return true
}

func Load[T Configurable](p string) (c T, err error) {
	k := koanf.New(".")
	f := file.Provider(p)
	err = k.Load(f, yaml.Parser())
	if err != nil {
		return
	}
	err = k.UnmarshalWithConf(
		"",
		&c,
		koanf.UnmarshalConf{
			Tag: "koanf",
			DecoderConfig: &mapstructure.DecoderConfig{
				DecodeHook: mapstructure.ComposeDecodeHookFunc(
					mapstructure.StringToTimeDurationHookFunc(),
					mapstructure.StringToSliceHookFunc(","),
					mapstructure.TextUnmarshallerHookFunc()),
				Metadata:         nil,
				Result:           &c,
				WeaklyTypedInput: true,
				Squash:           true,
			},
		},
	)

	if c.GetApp() == "" {
		err = kit.ErrConfigMissName
	}

	if c.GetEnvironment() == "" || !c.GetEnvironment().IsValid() {
		err = kit.ErrConfigMissEnvironment
	}

	if c.GetLogger() == nil {
		err = kit.ErrConfigMissLogger
	}

	return
}

// Sonyflake 创建sonyflake实例
func (c Configure) Sonyflake() (*sonyflake.Sonyflake, error) {
	id := 1103
	switch c.Environment {
	case kit.Production:
		id = 1101
	case kit.Development:
		id = 1102
	}
	return sonyflake.New(sonyflake.Settings{
		MachineID: func() (int, error) {
			return id, nil
		},
	})
}
