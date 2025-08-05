// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"bytes"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/labstack/echo/v4"
)

// Context Rest服务上下文
type Context struct {
	App string

	echo.Context
}

// NewContext 创建上下文
func NewContext(app string, c echo.Context) *Context {
	return &Context{
		App:     app,
		Context: c,
	}
}

// GetContext 获取上下文
func GetContext(c echo.Context) *Context {
	switch v := c.(type) {
	case *Context:
		return v
	default:
		return NewContext("UNKNOWN", c)
	}
}

// BindValidate 绑定并校验
func (c *Context) BindValidate(ptr any) {
	err := c.Bind(ptr)
	if err != nil {
		panic(NewError(http.StatusBadRequest, err.Error()))
	}
	err = c.Validate(ptr)
	if err != nil {
		panic(NewError(http.StatusBadRequest, err.Error()))
	}
}

// ContextBinding 获取上下文并绑定参数，返回 Context
func ContextBinding[T any](c echo.Context) (ctx *Context, req *T) {
	ctx = GetContext(c)
	req = new(T)
	ctx.BindValidate(req)
	return
}

// SendResponse 发送响应
func (c *Context) SendResponse(params ...any) error {
	buffer := &bytes.Buffer{}
	encoder := jsoniter.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(NewResponse[any]().SetParams(params...))

	return c.JSONBlob(http.StatusOK, buffer.Bytes())
}
