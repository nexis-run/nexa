// Copyright (C) nexa. 2026-present.
//
// Created at 2026-01-27, by liasica

package entx

import (
	"context"
	"fmt"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"entgo.io/ent/schema/mixin"
)

const SoftDeleteField = "deleted_at"

type softDeleteKey struct{}

type softDeleteFilter interface {
	WhereP(...func(*sql.Selector))
}

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

// Interceptors 默认过滤已软删除的记录
func (mixin SoftDeleteMixin) Interceptors() []ent.Interceptor {
	return []ent.Interceptor{
		SoftDeleteInterceptor(),
	}
}

// Hooks 禁止未显式授权的硬删除
func (SoftDeleteMixin) Hooks() []ent.Hook {
	return []ent.Hook{
		SoftDeleteHook(),
	}
}

// SkipSoftDelete 返回一个可查询已删除记录并允许硬删除的上下文
func SkipSoftDelete(parent context.Context) context.Context {
	return context.WithValue(parent, softDeleteKey{}, true)
}

func skipSoftDelete(ctx context.Context) bool {
	skip, _ := ctx.Value(softDeleteKey{}).(bool)

	return skip
}

func addSoftDeletePredicate(filter softDeleteFilter) {
	column := SoftDeleteField

	if metadata, ok := filter.(interface{ SoftDeleteColumn() string }); ok {
		column = metadata.SoftDeleteColumn()
	}

	filter.WhereP(sql.FieldIsNull(column))
}

// SoftDeleteInterceptor 软删除查询拦截器
func SoftDeleteInterceptor() ent.Interceptor {
	return ent.TraverseFunc(func(ctx context.Context, query ent.Query) (err error) {
		if skipSoftDelete(ctx) {
			return
		}

		filter, ok := query.(softDeleteFilter)
		if !ok {
			err = fmt.Errorf("%w：%T", ErrSoftDeleteQueryUnsupported, query)
			return
		}

		addSoftDeletePredicate(filter)

		return
	})
}

// SoftDeleteHook 禁止硬删除
func SoftDeleteHook() ent.Hook {
	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, mutation ent.Mutation) (value ent.Value, err error) {
			if skipSoftDelete(ctx) {
				value, err = next.Mutate(ctx, mutation)
				return
			}

			if mutation.Op().Is(ent.OpDelete | ent.OpDeleteOne) {
				err = ErrHardDeleteForbidden
				return
			}

			value, err = next.Mutate(ctx, mutation)

			return
		})
	}
}
