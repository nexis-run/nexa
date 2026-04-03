// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-25, by liasica

package parser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDIProvider(t *testing.T) {
	dp, err := NewDaoProvider("../../../../tests/di.gofile", "Dao", "daoProviderSet", "auroraride.com/oos/internal/infrastructure/dao")
	require.NoError(t, err)

	dp.AddField("Agreement", "Brand", "City", "Manager", "System")

	var b []byte

	b, err = dp.Generate()
	require.NoError(t, err)

	t.Logf("Generated DI Provider:\n%s", string(b))
}
