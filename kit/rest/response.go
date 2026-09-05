// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/labstack/echo/v4"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

func NewResponse() *Response {
	return &Response{
		Code:    http.StatusOK,
		Message: "",
	}
}

// SetCode 设置code
func (r *Response) SetCode(code int) *Response {
	r.Code = code
	return r
}

// SetMessage 设置message
func (r *Response) SetMessage(message string) *Response {
	r.Message = message
	return r
}

// SetData 设置data
// 仅过滤 nil 值, 数值零值 / 空串 / 空 struct 等有效零值原样保留
func (r *Response) SetData(data any) *Response {
	if data == nil {
		return r
	}

	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return r
	}

	// IsNil 仅对 chan/func/interface/map/pointer/slice 有效，其他类型会 panic
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if v.IsNil() {
			return r
		}
	}

	r.Data = data

	return r
}

// SetParams 设置响应参数
func (r *Response) SetParams(params ...any) *Response {
	for i := 0; i < len(params); i++ {
		switch v := params[i].(type) {
		case int:
			r.SetCode(v)
		case string:
			r.SetMessage(v)
		case error:
			var responseErr *Error
			var he *echo.HTTPError

			switch {
			case errors.As(v, &responseErr):
				if responseErr == nil {
					continue
				}

				r.SetCode(responseErr.Code).SetMessage(responseErr.Message)

				continue
			case errors.As(v, &he):
				if he == nil {
					continue
				}

				r.SetCode(he.Code)
				r.SetMessage(fmt.Sprint(he.Message))
			default:
				r.SetMessage(v.Error())
			}

			if r.Code == http.StatusOK {
				r.SetCode(http.StatusBadRequest)
			}
		default:
			if r.Data == nil {
				r.SetData(v)
			}
		}
	}

	if r.Message == "" {
		r.Message = http.StatusText(r.Code)
	}

	return r
}
