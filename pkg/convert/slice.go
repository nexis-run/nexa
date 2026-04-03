// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-11, by liasica

package convert

// Reverse 返回一个反转切片, 原切片不变
func Reverse[S ~[]E, E any](s S) (result S) {
	n := len(s)

	result = make(S, n)
	for i, j := 0, n-1; i < n; i, j = i+1, j-1 {
		result[i] = s[j]
	}

	return result
}
