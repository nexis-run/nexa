// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-11, by liasica

package convert

import "unsafe"

// UnsafeBytes2String 将字节切片无拷贝转换为字符串
func UnsafeBytes2String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

// UnsafeString2Bytes 将字符串无拷贝转换为字节切片
func UnsafeString2Bytes(s string) (b []byte) {
	return *(*[]byte)(unsafe.Pointer(&s))
}

// StringsToUint64 字符串切片转为 uint64 切片，忽略转换失败的值
func StringsToUint64(vals []string) []uint64 {
	data := make([]uint64, 0, len(vals))
	for _, v := range vals {
		u, ok := StringToUint64(v)
		if !ok {
			continue
		}

		data = append(data, u)
	}

	return data
}

// Uint64sToInterfaces 将 uint64 切片转换为 any 切片
func Uint64sToInterfaces(vals []uint64) []any {
	result := make([]any, len(vals))
	for i, v := range vals {
		result[i] = v
	}

	return result
}

// StringToUint64 解析字符串为 uint64，失败返回 false
func StringToUint64(s string) (uint64, bool) {
	if len(s) == 0 {
		return 0, false
	}

	var n uint64

	for _, ch := range []byte(s) {
		if ch < '0' || ch > '9' {
			return 0, false
		}

		n = n*10 + uint64(ch-'0')
	}

	return n, true
}
