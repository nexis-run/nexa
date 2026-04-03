// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"github.com/bytedance/sonic"
	"github.com/labstack/echo/v4"
)

var _ = (*DefaultJSONSerializer)(nil)

// DefaultJSONSerializer implements JSON encoding using encoding/jsoniter.
type DefaultJSONSerializer struct{}

func NewDefaultJSONSerializer() *DefaultJSONSerializer {
	return &DefaultJSONSerializer{}
}

// Serialize converts an interface into a json and writes it to the response.
// You can optionally use the indent parameter to produce pretty JSONs.
func (d DefaultJSONSerializer) Serialize(c echo.Context, i any, indent string) error {
	enc := sonic.ConfigDefault.NewEncoder(c.Response())

	if indent != "" {
		enc.SetIndent("", indent)
	}

	return enc.Encode(i)
}

// Deserialize reads a JSON from a request body and converts it into an interface.
func (d DefaultJSONSerializer) Deserialize(c echo.Context, i any) error {
	err := sonic.ConfigDefault.NewDecoder(c.Request().Body).Decode(i)
	return err
}
