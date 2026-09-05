// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-26, by liasica

package pool

import (
	"bytes"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var l = 10000

func TestPoolZeroValueAndFactory(t *testing.T) {
	var values Pool[int]
	require.Zero(t, values.Get())
	require.Nil(t, NewPool[any](nil).Get())
	require.Nil(t, NewPool[any](func() any { return nil }).Get())
	require.Equal(t, "new", NewPool(func() string { return "new" }).Get())
	values.Put(7)
	value := values.Get()
	require.Contains(t, []int{0, 7}, value)
}

func fillBuffer(b *bytes.Buffer) {
	for i := 0; i < l; i++ {
		b.WriteByte(0)
	}
}

func BenchmarkBuffer(b *testing.B) {
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		wg := &sync.WaitGroup{}

		for i := 0; i < 1000; i++ {
			wg.Add(1)

			go func(wg *sync.WaitGroup) {
				defer wg.Done()

				fillBuffer(bytes.NewBuffer(nil))
			}(wg)
		}
		wg.Wait()
	}
}

func BenchmarkBufferPool(b *testing.B) {
	b.ReportAllocs()

	for n := 0; n < b.N; n++ {
		wg := &sync.WaitGroup{}

		for i := 0; i < 1000; i++ {
			wg.Add(1)

			go func(wg *sync.WaitGroup) {
				defer wg.Done()

				buf := GetBuffer()
				fillBuffer(buf)
				PutBuffer(buf)
			}(wg)
		}

		wg.Wait()
	}
}
