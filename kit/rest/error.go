// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-27, by liasica

package rest

import (
	"net/http"
	"strconv"
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *Error) Error() string {
	return "code: " + strconv.Itoa(e.Code) + ", message: " + e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
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

func WrapError(code int, err error) *Error {
	if err == nil {
		return NewError(code, "")
	}

	e := NewError(code, err.Error())

	e.Err = err

	return e
}
