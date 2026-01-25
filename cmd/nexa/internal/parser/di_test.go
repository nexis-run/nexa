// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-25, by liasica

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDIProvider(t *testing.T) {
	dp, err := ParseDaoProvider("../../../../tests/di.gofile", "Dao", "daoProviderSet")
	require.NoError(t, err)

	_, err = dp.Generate()
	require.NoError(t, err)
}
