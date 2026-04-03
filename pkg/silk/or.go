// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-23, by liasica

package silk

func Or[T any](condition bool, yes T, no T) T {
	if condition {
		return yes
	}

	return no
}

func OrFunc[T any](condition func() bool, yes func() T, no func() T) T {
	if condition() {
		return yes()
	}

	return no()
}
