// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-25, by liasica

package convert

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReverse(t *testing.T) {
	s := []int{1, 2, 3, 4, 5}
	reversed := Reverse(s)
	expected := []int{5, 4, 3, 2, 1}
	require.EqualValues(t, expected, reversed)

	slices.Reverse(s)
	require.EqualValues(t, expected, s)
}

func BenchmarkReverse(b *testing.B) {
	s := make([]int, 1000)
	for i := range s {
		s[i] = i
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = Reverse(s)
	}
}

func TestStringsToUint64(t *testing.T) {
	vals := []string{"123", "456", "abc", "789"}
	expected := []uint64{123, 456, 789}
	result := StringsToUint64(vals)
	require.EqualValues(t, expected, result)

	anyTypes := Uint64sToInterfaces(result)
	require.EqualValues(t, []any{uint64(123), uint64(456), uint64(789)}, anyTypes)
}
