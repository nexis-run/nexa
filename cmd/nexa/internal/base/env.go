// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-20, by liasica

package base

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
	"unicode"
)

func StringIsUpperStart(name string) bool {
	if name == "" {
		return false
	}

	return unicode.IsUpper(rune(name[0]))
}

func GetDate() string {
	return time.Now().Format("2006-01-02")
}

func GetYear() string {
	return strconv.Itoa(time.Now().Year())
}

// GetPkgPath 获取包的 import 路径
func GetPkgPath(module, root, path string) string {
	adir, _ := filepath.Abs(root)

	apath := path
	if !filepath.IsAbs(apath) {
		apath, _ = filepath.Abs(filepath.Join(adir, path))
	}

	rel, _ := filepath.Rel(adir, apath)
	result := filepath.ToSlash(filepath.Join(module, rel))

	return result
}

func MkdirAll(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModePerm)
	}

	return nil
}
