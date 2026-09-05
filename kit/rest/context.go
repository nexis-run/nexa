// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"bytes"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/labstack/echo/v4"
	"gopkg.auroraride.com/rbac"
)

const (
	ContextKeyUser = "_user"
)

// Context Rest服务上下文
type Context struct {
	App string

	echo.Context

	User *rbac.User
}

// NewContext 创建上下文
func NewContext(app string, c echo.Context) *Context {
	user, _ := c.Get(ContextKeyUser).(*rbac.User)

	return &Context{
		App:     app,
		Context: c,
		User:    user,
	}
}

// NexaContext 返回基础上下文，支持嵌入 Context 的自定义类型
func (c *Context) NexaContext() *Context {
	return c
}

// GetContext 获取上下文
func GetContext(c echo.Context) *Context {
	if carrier, ok := c.(interface{ NexaContext() *Context }); ok {
		return carrier.NexaContext()
	}

	return NewContext("UNKNOWN", c)
}

func ensureContext(c echo.Context) echo.Context {
	if _, ok := c.(interface{ NexaContext() *Context }); ok {
		return c
	}

	return GetContext(c)
}

// BindValidate 绑定并校验（失败时 panic，需配合 RecoverMiddleware 使用）
func (c *Context) BindValidate(ptr any) {
	if err := c.BindAndValidate(ptr); err != nil {
		panic(err)
	}
}

// BindAndValidate 绑定并校验，返回 error 而非 panic
func (c *Context) BindAndValidate(ptr any) error {
	err := c.Bind(ptr)
	if err != nil {
		return NewError(http.StatusBadRequest, err.Error())
	}

	err = c.Validate(ptr)
	if err != nil {
		return NewError(http.StatusBadRequest, err.Error())
	}

	return nil
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
	encoder := sonic.ConfigDefault.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)

	response := NewResponse().SetParams(params...)
	if response.Code < http.StatusOK || response.Code > 599 {
		return NewError(http.StatusInternalServerError, "无效的 HTTP 响应状态码")
	}

	if response.Code == http.StatusNoContent || response.Code == http.StatusNotModified {
		return c.NoContent(response.Code)
	}

	if err := encoder.Encode(response); err != nil {
		return err
	}

	return c.JSONBlob(response.Code, buffer.Bytes())
}
