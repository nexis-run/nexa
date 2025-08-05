// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

func NewError(code int, message string) *Error {
	err := &Error{
		Code:    code,
		Message: message,
	}
	if err.Message == "" {
		err.Message = http.StatusText(code)
	}
	return err
}

type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    *T     `json:"data,omitempty"`
}

func NewResponse[T any]() *Response[T] {
	return &Response[T]{
		Code:    http.StatusOK,
		Message: "",
	}
}

// SetCode 设置code
func (r *Response[T]) SetCode(code int) *Response[T] {
	r.Code = code
	return r
}

// SetMessage 设置message
func (r *Response[T]) SetMessage(message string) *Response[T] {
	r.Message = message
	return r
}

// SetData 设置data
func (r *Response[T]) SetData(data *T) *Response[T] {
	r.Data = data
	return r
}

// SetParams 设置响应参数
func (r *Response[T]) SetParams(params ...any) *Response[T] {
	for i := 0; i < len(params); i++ {
		switch v := params[i].(type) {
		case int:
			r.SetCode(v)
		case string:
			r.SetMessage(v)
		case *Error:
			r.SetCode(v.Code).SetMessage(v.Message)
		case error:
			message := v.Error()
			var he *echo.HTTPError
			if errors.As(v, &he) {
				message = fmt.Sprintf("%v", he.Message)
			}
			r.SetMessage(message)
		case *T:
			if r.Data == nil {
				r.SetData(v)
			}
			// default:
			// 	if r.Data == nil {
			// 		r.SetData(v)
			// 	}
		}
	}

	if r.Message == "" {
		r.Message = http.StatusText(r.Code)
	}

	return r
}
