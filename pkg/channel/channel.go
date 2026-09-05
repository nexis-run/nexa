// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-22, by liasica

package channel

// SafeClose 尝试关闭通道，nil 或已关闭时返回 false
// 调用方必须先停止所有发送者，不能将此函数与发送操作并发调用
func SafeClose[T any](ch chan T) (justClosed bool) {
	defer func() {
		if recover() != nil {
			justClosed = false
		}
	}()

	if ch == nil {
		return
	}

	close(ch)

	justClosed = true

	return
}

// SafeSend 阻塞发送一个值，nil 或已关闭时返回 true
// panic 恢复不提供发送与关闭之间的同步，调用方必须协调通道所有权
func SafeSend[T any](ch chan T, value T) (closed bool) {
	if ch == nil {
		closed = true
		return
	}

	defer func() {
		if recover() != nil {
			closed = true
		}
	}()

	ch <- value

	return
}
