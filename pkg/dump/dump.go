// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-19, by liasica

package dump

import "strings"

const hextable = "0123456789ABCDEF"

// Bytes 将字节切片转换为十六进制字符串表示
func Bytes(src []byte) string {
	if len(src) == 0 {
		return ""
	}

	// 每个字节占 2 个十六进制字符 + 1 个空格分隔符（最后一个除外）
	var buf strings.Builder
	buf.Grow(len(src)*3 - 1)

	for i, b := range src {
		if i > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteByte(hextable[b>>4])
		buf.WriteByte(hextable[b&0x0f])
	}

	return buf.String()
}
