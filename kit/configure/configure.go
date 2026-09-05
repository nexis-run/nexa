// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-25, by liasica

package configure

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"gopkg.in/yaml.v3"

	"nexis.run/nexa/kit"
)

type Configure struct {
	App                string          // 应用名称
	Environment        kit.Environment // 环境变量
	Logger             *Logger         // 日志配置
	SonyflakeMachineID *int            `koanf:"sonyflake_machine_id"` // 同一 ID 空间内唯一的进程编号，范围为 0~65535
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

	// 输出至 Kafka
	Kafka *LoggerKafka
}

type LoggerKafka struct {
	Disable bool     // 是否禁用 Kafka 日志输出
	Topic   string   // Kafka topic
	Brokers []string // Kafka brokers
}

func (l *Logger) IsValid() (valid bool) {
	if l == nil {
		return
	}

	if l.Kafka == nil || l.Kafka.Disable {
		return l.Stdout
	}

	if strings.TrimSpace(l.Kafka.Topic) == "" || len(l.Kafka.Brokers) == 0 {
		return
	}

	for _, broker := range l.Kafka.Brokers {
		if strings.TrimSpace(broker) == "" {
			return
		}
	}

	return true
}

// Load 读取单个 YAML 配置，使用 koanf 标签、嵌入字段和类型转换钩子
func Load[T Configurable](p string) (c T, err error) {
	var content []byte

	content, err = os.ReadFile(p)
	if err != nil {
		err = fmt.Errorf("读取配置 %s 失败：%w", p, err)
		return
	}

	var values map[string]any
	parser := yaml.NewDecoder(bytes.NewReader(content))

	err = parser.Decode(&values)
	if err != nil {
		err = fmt.Errorf("解析配置 %s 失败：%w", p, err)
		return
	}

	if len(values) == 0 {
		err = kit.ErrConfigMissName
		return
	}

	var extra any

	err = parser.Decode(&extra)
	if !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("配置文件只能包含一个 YAML 文档")
		}

		return
	}

	var decoder *mapstructure.Decoder

	decoder, err = mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "koanf",
		Result:  &c,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			validateIntegerConversion,
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			mapstructure.TextUnmarshallerHookFunc()),
		WeaklyTypedInput: true,
		Squash:           true,
	})
	if err != nil {
		return
	}

	err = decoder.Decode(values)
	if err != nil {
		return
	}

	if strings.TrimSpace(c.GetApp()) == "" {
		err = kit.ErrConfigMissName
		return
	}

	if c.GetEnvironment() == "" || !c.GetEnvironment().IsValid() {
		err = kit.ErrConfigMissEnvironment
		return
	}

	if c.GetLogger() == nil {
		err = kit.ErrConfigMissLogger
		return
	}

	if !c.GetLogger().IsValid() {
		err = kit.ErrConfigInvalidLogger
	}

	return
}

// validateIntegerConversion 拒绝浮点数到整数字段的截断和溢出
func validateIntegerConversion(from, to reflect.Value) (any, error) {
	if from.Kind() != reflect.Float32 && from.Kind() != reflect.Float64 {
		return from.Interface(), nil
	}

	kind := to.Kind()
	if kind < reflect.Int || kind > reflect.Uint64 {
		return from.Interface(), nil
	}

	number := from.Float()
	upper := math.Ldexp(1, to.Type().Bits())
	lower := 0.0

	if kind <= reflect.Int64 {
		upper /= 2
		lower = -upper
	}

	if math.IsNaN(number) || math.Trunc(number) != number || number < lower || number >= upper {
		return nil, fmt.Errorf("%v 不能无损转换为 %s", number, to.Type())
	}

	return from.Interface(), nil
}

// IsVaild 检查日志配置是否合法
func (l *Logger) IsVaild() bool {
	return l.IsValid()
}
