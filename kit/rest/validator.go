// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-04, by liasica

package rest

import (
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
	// if err := v.validator.Struct(i); err != nil {
	// 	return NewError(http.StatusBadRequest, err.Error())
	// }
	// return nil
	return v.validator.Struct(i)
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
			return err
		}

		return v.validator.RegisterTranslation(
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
	}
}
