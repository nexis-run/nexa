// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-27, by liasica

package entgen

import (
	"path/filepath"
	"sort"

	"entgo.io/ent/entc"
	"entgo.io/ent/entc/gen"
	"entgo.io/ent/schema/field"
)

var (
	defaultFeats = []string{"sql/modifier", "sql/upsert", "privacy", "entql", "sql/execquery", "intercept", "schema/snapshot"}
)

type GenerateExtension struct {
}

func (g *GenerateExtension) Hooks() []gen.Hook {
	return nil
}

func (g *GenerateExtension) Annotations() []entc.Annotation {
	return nil
}

func (g *GenerateExtension) Templates() []*gen.Template {
	return []*gen.Template{
		gen.MustParse(gen.NewTemplate("meta").Parse(TemplateMeta)),
		gen.MustParse(gen.NewTemplate("soft_delete").Parse(TemplateSoftDelete)),
		gen.MustParse(gen.NewTemplate("upsert").Parse(TemplateUpsert)),
	}
}

func (g *GenerateExtension) Options() []entc.Option {
	return nil
}

func (eng *EntGen) Generate() error {
	// 特性列表
	fm := make(map[string]struct{})

	for _, feat := range append(defaultFeats, eng.cfg.EntFeatures...) {
		fm[feat] = struct{}{}
	}

	var features []string

	for feat := range fm {
		features = append(features, feat)
	}
	sort.Strings(features)

	// 模板列表
	templates := []*gen.Template{
		gen.MustParse(gen.NewTemplate("meta").Parse(TemplateMeta)),
		gen.MustParse(gen.NewTemplate("soft_delete").Parse(TemplateSoftDelete)),
		gen.MustParse(gen.NewTemplate("upsert").Parse(TemplateUpsert)),
	}

	opts := []entc.Option{
		entc.Storage("sql"),
		entc.FeatureNames(features...),
	}

	cfg := gen.Config{
		IDType:    &field.TypeInfo{Type: field.TypeUint64},
		Package:   eng.cfg.GetEntPkgPath(),
		Templates: templates,
	}

	p, err := eng.cfg.GetEntPath()
	if err != nil {
		return err
	}

	return entc.Generate(filepath.Join(p, "schema"), &cfg, opts...)
}
