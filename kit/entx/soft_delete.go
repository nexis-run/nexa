// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-27, by liasica

package entx

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

const SoftDeleteField = "deleted_at"

// SoftDeleteMixin 软删除混入
type SoftDeleteMixin struct {
	mixin.Schema
}

// Fields 定义软删除字段
func (SoftDeleteMixin) Fields() []ent.Field {
	return []ent.Field{
		field.Time(SoftDeleteField).Nillable().Optional(),
	}
}

// Indexes 定义软删除索引
func (SoftDeleteMixin) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields(SoftDeleteField),
	}
}

// SoftDeleteInterceptor 软删除查询拦截器
func SoftDeleteInterceptor() ent.Interceptor {
	return ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			selector, ok := q.(*sql.Selector)
			if ok {
				selector.Where(
					sql.IsNull(selector.C(SoftDeleteField)),
				)
			}

			return next.Query(ctx, q)
		})
	})
}

// SoftDeleteHook 禁止硬删除
func SoftDeleteHook() ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if m.Op().Is(ent.OpDelete | ent.OpDeleteOne) {
				return nil, ErrHardDeleteForbidden
			}

			return next.Mutate(ctx, m)
		})
	}
}
