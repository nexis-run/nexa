// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-20, by liasica

package base

// DaoTemplateVariables 定义 DAO 模板变量
type DaoTemplateVariables struct {
	Package      string
	EntPkgImport string
	Name         string
	OrmClient    string
}

// EchoCtxTemplateVariables 定义 echo Context 模板变量
type EchoCtxTemplateVariables struct {
	Package string
	Name    string
}
