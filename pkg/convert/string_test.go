// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-25, by liasica

package convert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnsafeString(t *testing.T) {
	str := "hello world"
	b := []byte(str)
	s := UnsafeBytes2String(b)
	require.Equal(t, s, str)

	b2 := UnsafeString2Bytes(s)
	require.Equal(t, b2, b)
	require.Equal(t, len(s), cap(b2))
}
