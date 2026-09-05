// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
	"errors"
	"strings"

	zhLocale "github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTranslation "github.com/go-playground/validator/v10/translations/zh"
)

type Validator struct {
	validator *validator.Validate
	trans     ut.Translator
}

type RegisterValidationFunc func(fn validator.Func) (err error)

func NewValidator() *Validator {
	zh := zhLocale.New()
	uni := ut.New(zh, zh)
	trans, _ := uni.GetTranslator("zh")

	validate := validator.New()

	_ = zhTranslation.RegisterDefaultTranslations(validate, trans)

	return &Validator{validator: validate, trans: trans}
}

func (v *Validator) Validate(i any) error {
	return v.validator.Struct(i)
}

// Translate 按字段顺序输出校验消息，支持默认中文与自定义翻译
func (v *Validator) Translate(err error) string {
	if err == nil {
		return ""
	}

	var fields validator.ValidationErrors
	if !errors.As(err, &fields) {
		return err.Error()
	}

	messages := make([]string, 0, len(fields))
	for _, field := range fields {
		messages = append(messages, field.Translate(v.trans))
	}

	return strings.Join(messages, "；")
}

// Validator 获取底层 validator 实例
func (v *Validator) Validator() *validator.Validate {
	return v.validator
}

// RegisterValidation 注册自定义校验方法
func (v *Validator) RegisterValidation(tag string, message ...string) RegisterValidationFunc {
	return func(fn validator.Func) (err error) {
		err = v.validator.RegisterValidation(tag, fn)
		if err != nil {
			return
		}

		err = v.validator.RegisterTranslation(
			tag,
			v.trans,
			func(ut ut.Translator) error {
				text := "{0}验证失败"

				if len(message) > 0 {
					text = message[0]
				}

				return ut.Add(tag, text, true)
			}, func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T(tag, fe.Field())
				return t
			},
		)

		return
	}
}
