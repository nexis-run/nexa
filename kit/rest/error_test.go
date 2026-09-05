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

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestResponseErrorEntrypoints(t *testing.T) {
	cases := []struct {
		failure error
		status  int
		message string
	}{
		{errors.New("private database details"), http.StatusInternalServerError, "Internal Server Error"},
		{NewError(http.StatusInternalServerError, "private server details"), http.StatusInternalServerError, "Internal Server Error"},
		{status.Error(codes.NotFound, "record missing"), http.StatusNotFound, "record missing"},
		{NewError(http.StatusConflict, "record exists"), http.StatusConflict, "record exists"},
	}

	for _, test := range cases {
		for _, direct := range []bool{false, true} {
			server := New("test", nil)
			server.GET("/", func(c echo.Context) error {
				if direct {
					return GetContext(c).SendResponse(map[string]string{"secret": "partial data"}, test.failure)
				}

				return test.failure
			})

			response := httptest.NewRecorder()
			server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
			require.Equal(t, test.status, response.Code)
			require.Contains(t, response.Body.String(), test.message)
			require.NotContains(t, response.Body.String(), "private")
			require.NotContains(t, response.Body.String(), "partial data")
		}
	}

	var empty *Error
	server := New("test", nil)
	require.NotPanics(t, func() {
		handleHTTPError(empty, server.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder()))
	})
}

func TestBindValidateUsesCustomTranslation(t *testing.T) {
	server := New("test", nil)
	validation := NewValidator()
	require.NoError(t, validation.RegisterValidation("allowed", "{0}不在允许范围内")(func(validator.FieldLevel) bool { return false }))
	server.Validator = validation

	type request struct {
		Name string `json:"name" validate:"allowed"`
	}

	incoming := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"example"}`))
	incoming.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	ctx := NewContext("test", server.NewContext(incoming, httptest.NewRecorder()))
	err := ctx.BindAndValidate(&request{})
	require.ErrorContains(t, err, "Name不在允许范围内")

	var fields validator.ValidationErrors
	require.ErrorAs(t, err, &fields)
}
