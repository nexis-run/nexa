// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-18, by liasica

package pool

import (
	"bytes"
	"sync"
)

// buffer pool to reduce GC
var buffers = sync.Pool{
	// New is called when a new instance is needed
	New: func() any {
		return new(bytes.Buffer)
	},
}

// GetBuffer fetches a buffer from the pool
func GetBuffer() *bytes.Buffer {
	return buffers.Get().(*bytes.Buffer)
}

// PutBuffer returns a buffer to the pool
func PutBuffer(buf *bytes.Buffer) {
	// See https://golang.org/issue/23199
	const maxSize = 1 << 16
	if buf != nil && buf.Cap() < maxSize {
		buf.Reset()
		buffers.Put(buf)
	}
}

// Pool 保存同一类型的临时对象，零值可用，首次使用后不能复制
type Pool[T any] struct {
	pool sync.Pool
}

// Get 获取对象，池为空且未设置工厂时返回 T 的零值
func (p *Pool[T]) Get() (value T) {
	item := p.pool.Get()
	if item != nil {
		value = item.(T)
	}

	return
}

func (p *Pool[T]) Put(x T) {
	p.pool.Put(x)
}

func NewPool[T any](f func() T) *Pool[T] {
	result := &Pool[T]{}

	if f != nil {
		result.pool.New = func() any { return f() }
	}

	return result
}
