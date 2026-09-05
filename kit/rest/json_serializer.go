// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"github.com/bytedance/sonic"
	"github.com/labstack/echo/v4"
)

var _ echo.JSONSerializer = (*DefaultJSONSerializer)(nil)

// DefaultJSONSerializer 使用 Sonic 编解码 JSON
type DefaultJSONSerializer struct{}

func NewDefaultJSONSerializer() *DefaultJSONSerializer {
	return &DefaultJSONSerializer{}
}

// Serialize 将 JSON 写入响应，indent 非空时使用缩进
func (d DefaultJSONSerializer) Serialize(c echo.Context, i any, indent string) error {
	enc := sonic.ConfigDefault.NewEncoder(c.Response())

	if indent != "" {
		enc.SetIndent("", indent)
	}

	return enc.Encode(i)
}

// Deserialize 将请求正文解析到目标对象
func (d DefaultJSONSerializer) Deserialize(c echo.Context, i any) error {
	return sonic.ConfigDefault.NewDecoder(c.Request().Body).Decode(i)
}
