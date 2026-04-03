// Copyright (C) micros. 2024-present.
//
// Created at 2024-12-19, by liasica

package dump

const hextable = "0123456789ABCDEF"

func Bytes(src []byte) (str string) {
	for i := 0; i < len(src); i++ {
		str += string(hextable[src[i]>>4]) + string(hextable[src[i]&0x0f])
		if i != len(src)-1 {
			str += " "
		}
	}

	return
}
