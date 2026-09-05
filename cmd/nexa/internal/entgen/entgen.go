package entgen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
)

type EntGen struct {
	cfg *base.Config
}

func New(cfg *base.Config) (entGen *EntGen, err error) {
	if cfg == nil {
		err = fmt.Errorf("生成 Ent 配置不能为空")
		return
	}

	_, err = cfg.ResolveModule()
	if err != nil {
		return
	}

	entGen = &EntGen{cfg: cfg}

	return
}

func (eng *EntGen) planGenerateFile(target string) (files []fileplan.File, err error) {
	path := filepath.Join(target, "generate.go")
	var entries []os.DirEntry

	entries, err = os.ReadDir(target)
	if err != nil && !os.IsNotExist(err) {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		var current []byte

		current, err = os.ReadFile(filepath.Join(target, entry.Name()))
		if err != nil {
			return
		}

		if entry.Name() == "generate.go" || bytes.Contains(current, []byte("//go:generate")) {
			return
		}
	}

	directive := "//go:generate nexa ent generate"

	configPath := eng.cfg.GetConfigFilePath()
	if configPath != "" {
		_, err = os.Stat(configPath)
		if err == nil {
			var relativeConfig string

			relativeConfig, err = filepath.Rel(target, configPath)
			if err != nil {
				return
			}

			directive = "//go:generate nexa --config " + strconv.Quote(filepath.ToSlash(relativeConfig)) + " ent generate"
		} else if !os.IsNotExist(err) {
			return
		}
	}

	var content []byte

	content, err = format.Source(fmt.Appendf(nil, "package %s\n\n%s\n", filepath.Base(target), directive))
	if err != nil {
		err = fmt.Errorf("生成 Ent 包名称无效：%w", err)
		return
	}

	files = []fileplan.File{{Path: path, Content: content}}

	return
}
