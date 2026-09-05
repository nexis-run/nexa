// Copyright (C) nexa. 2026-present.
//
// Created at 2026-09-03, by liasica

package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestDumpPreservesAndLimitsBodies(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("request-body"))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var capturedRequestBody []byte
	var capturedResponseBody []byte
	middleware := dump(&DumpConfig{
		RequestBody:  true,
		ResponseBody: true,
		BodyMaxBytes: 4,
	}, func(_ echo.Context, requestBody, responseBody []byte) {
		capturedRequestBody = append(capturedRequestBody, requestBody...)
		capturedResponseBody = append(capturedResponseBody, responseBody...)
	})

	err := middleware(func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		require.NoError(t, err)
		require.Equal(t, "request-body", string(body))

		return c.String(http.StatusOK, "response-body")
	})(c)

	require.NoError(t, err)
	require.Equal(t, "requ", string(capturedRequestBody))
	require.Equal(t, "resp", string(capturedResponseBody))
	require.Equal(t, "response-body", rec.Body.String())
	require.Equal(t, true, c.Get(dumpRequestBodyTruncatedKey))
	require.Equal(t, true, c.Get(dumpResponseBodyTruncatedKey))
}

func TestDumpCapturesErrorResponse(t *testing.T) {
	server := New("test", nil)
	var captured []byte
	server.Use(dump(&DumpConfig{ResponseBody: true, BodyMaxBytes: 1024}, func(_ echo.Context, _, response []byte) {
		captured = append(captured, response...)
	}))
	server.GET("/", func(echo.Context) error { return NewError(http.StatusNotFound, "record missing") })

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, response.Body.Bytes(), captured)
	require.Contains(t, string(captured), "record missing")
}
