// Copyright (C) nexa. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"net/http"
	"reflect"
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
// 仅过滤 nil 值，数值零值、空串和空结构体原样保留
func (r *Response) SetData(data any) *Response {
	if !isNil(data) {
		r.Data = data
	}

	return r
}

// isNil 判断接口值本身或其承载的指针类值是否为 nil
func isNil(value any) bool {
	if value == nil {
		return true
	}

	// IsNil 仅对 chan/func/interface/map/pointer/slice 有效，其他类型会 panic
	switch reflected := reflect.ValueOf(value); reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

// SetParams 设置响应参数
func (r *Response) SetParams(params ...any) *Response {
	var failure *Response

	for _, param := range params {
		switch value := param.(type) {
		case int:
			r.SetCode(value)
		case string:
			r.SetMessage(value)
		case error:
			if response := responseFromError(value); response != nil {
				failure = response
			}
		default:
			if r.Data == nil {
				r.SetData(value)
			}
		}
	}

	if failure != nil {
		*r = *failure
	}

	if r.Message == "" {
		r.Message = http.StatusText(r.Code)
	}

	return r
}
