// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-11, by liasica

package convert

import "slices"

// Reverse 返回反转后的副本，保留 nil 与空切片的区别
func Reverse[S ~[]E, E any](s S) S {
	result := slices.Clone(s)
	slices.Reverse(result)

	return result
}
