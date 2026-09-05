// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-27, by liasica

package rest

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gopkg.auroraride.com/rbac"

	"nexis.run/nexa/kit/authz"
)

func TestWrapError(t *testing.T) {
	err := WrapError(http.StatusUnauthorized, authz.ErrUnauthorized)
	require.ErrorIs(t, err, authz.ErrUnauthorized)
}

func TestSendResponseUsesHTTPStatus(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := NewContext("test", e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec))

	err := c.SendResponse(NewError(http.StatusBadRequest, "请求错误"))

	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleHTTPErrorHidesInternalDetails(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

	handleHTTPError(errors.New("数据库密码泄露"), c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.False(t, strings.Contains(rec.Body.String(), "数据库密码泄露"))
}

type testContext struct {
	*Context
}

func TestMiddlewareKeepsCustomContext(t *testing.T) {
	user := &rbac.User{Uid: "test"}
	middlewares := []echo.MiddlewareFunc{
		RecoverMiddleware(),
		RBACMiddleware(WithRBACRemoteAuth(false), WithRBACStaticUser(user)),
		RBACMiddleware(WithRBACSkipper(func(echo.Context) bool { return true })),
	}

	for _, middleware := range middlewares {
		e := echo.New()
		ctx := &testContext{Context: NewContext("test", e.NewContext(
			httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(),
		))}
		handler := middleware(func(c echo.Context) error {
			require.Same(t, ctx, c)
			return nil
		})

		require.NoError(t, handler(ctx))
	}
}

func TestWrappedResponseError(t *testing.T) {
	err := fmt.Errorf("处理失败：%w", NewError(http.StatusConflict, "存在冲突"))
	response := NewResponse().SetParams(err)
	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, "存在冲突", response.Message)

	var empty *Error
	response = NewResponse().SetParams(empty)
	require.Equal(t, http.StatusOK, response.Code)

	e := New("test", nil)
	e.GET("/", func(echo.Context) error { panic(err) })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusConflict, rec.Code)
}

func TestRecoverKeepsHTTPAbort(t *testing.T) {
	e := New("test", nil)
	e.GET("/", func(echo.Context) error { panic(http.ErrAbortHandler) })

	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})
}

func TestMissingHTMLTemplate(t *testing.T) {
	templates := &HTMLTemplate{}
	require.Error(t, templates.Render(io.Discard, "missing.html", nil, nil))
}

func TestGetRequestURL(t *testing.T) {
	cases := []struct {
		header string
		value  string
		want   string
	}{
		{header: "X-Original-URL", value: "https://public.example/a%2Fb?x=1", want: "https://public.example/a%2Fb?x=1"},
		{header: "X-Original-URI", value: "/origin%2Fpath?x=1", want: "http://internal/origin%2Fpath?x=1"},
		{header: "X-Forwarded-Prefix", value: "/proxy", want: "http://internal/proxy/a%2Fb?x=1"},
		{want: "http://internal/a%2Fb?x=1"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "http://internal/a%2Fb?x=1", nil)

		if tc.header != "" {
			req.Header.Set(tc.header, tc.value)
		}

		ctx := echo.New().NewContext(req, httptest.NewRecorder())
		actual, err := GetRequestURL(ctx)
		require.NoError(t, err)
		require.Equal(t, tc.want, actual.String())
	}
}
