// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-27, by liasica

package entx

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

// TimeMixin 时间字段
type TimeMixin struct {
	mixin.Schema

	DisableIndex bool
	Optional     bool
}

func (t TimeMixin) Fields() []ent.Field {
	creator := field.Time("created_at").Immutable()
	updator := field.Time("updated_at")

	if t.Optional {
		creator.Optional().Nillable()
		updator.Optional().Nillable()
	}

	return []ent.Field{
		creator.Default(time.Now),
		updator.Default(time.Now).UpdateDefault(time.Now),
	}
}

func (t TimeMixin) Indexes() (indexes []ent.Index) {
	if !t.DisableIndex {
		indexes = append(indexes, index.Fields("created_at"), index.Fields("updated_at"))
	}

	return
}
