// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-22, by liasica

package channel

func SafeClose[T any](ch chan T) (justClosed bool) {
	defer func() {
		if recover() != nil {
			// The return result can be altered
			// in a defer function call.
			justClosed = false
		}
	}()

	// assume ch != nil here.
	close(ch) // panic if ch is closed

	return true // <=> justClosed = true; return
}

func SafeSend[T any](ch chan T, value T) (closed bool) {
	defer func() {
		if recover() != nil {
			closed = true
		}
	}()

	ch <- value // panic if ch is closed

	return false // <=> closed = false; return
}
