// Copyright (C) nexa. 2026-present.
//
// Created at 2026-09-03, by liasica

package entx

import (
	"context"
	"testing"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/stretchr/testify/require"
)

type softDeleteTestQuery struct {
	predicates []func(*sql.Selector)
}

func (query *softDeleteTestQuery) WhereP(predicates ...func(*sql.Selector)) {
	query.predicates = append(query.predicates, predicates...)
}

func TestSoftDeleteInterceptor(t *testing.T) {
	interceptor := SoftDeleteInterceptor()
	traverser, ok := interceptor.(ent.Traverser)
	require.True(t, ok)

	query := &softDeleteTestQuery{}
	err := traverser.Traverse(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, query.predicates, 1)

	skippedQuery := &softDeleteTestQuery{}
	err = traverser.Traverse(SkipSoftDelete(context.Background()), skippedQuery)
	require.NoError(t, err)
	require.Empty(t, skippedQuery.predicates)

	err = traverser.Traverse(context.Background(), struct{}{})
	require.ErrorIs(t, err, ErrSoftDeleteQueryUnsupported)
}

func TestSoftDeleteMixinRegistersMiddleware(t *testing.T) {
	mixin := SoftDeleteMixin{}

	require.Len(t, mixin.Interceptors(), 1)
	require.Len(t, mixin.Hooks(), 1)
}
