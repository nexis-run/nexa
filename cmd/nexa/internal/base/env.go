// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-20, by liasica

package base

import (
	"go/ast"
	"go/token"
)

func StringIsExportedIdentifier(name string) bool {
	return token.IsIdentifier(name) && ast.IsExported(name)
}
