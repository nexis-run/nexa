// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-26, by liasica

package pool

import (
	"bytes"
	"sync"
	"testing"
)

var l = 10000

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
