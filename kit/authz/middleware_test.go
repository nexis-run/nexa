// Copyright (C) nexa. 2026-present.
//
// Created at 2026-09-06, by liasica

package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gopkg.auroraride.com/rbac"

	"nexis.run/nexa/kit/rest"
)

func TestMiddleware(t *testing.T) {
	client, err := New(startTestServer(t))
	require.NoError(t, err)

	t.Cleanup(func() { _ = client.Close() })

	server := echo.New()
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, UserFromContext(c).GetUid())
	}

	cases := []struct {
		name       string
		middleware echo.MiddlewareFunc
		token      string
		status     int
		uid        string
	}{
		{"remote allowed", Middleware(WithClient(client), WithProjectCode(existingProject), WithPermissionKey(allowedPermission)), testToken, http.StatusOK, testUID},
		{"remote forbidden", Middleware(WithClient(client), WithProjectCode(existingProject), WithPermissionKey("forbidden_permission")), testToken, http.StatusForbidden, ""},
		{"remote unknown user", Middleware(WithClient(client), WithProjectCode("non_existing_project"), WithPermissionKey(allowedPermission)), testToken, http.StatusUnauthorized, ""},
		{"missing token", Middleware(WithClient(client)), "", http.StatusUnauthorized, ""},
		{"static user", Middleware(WithRemoteAuth(false), WithStaticUser(&rbac.User{Uid: testUID})), "", http.StatusOK, testUID},
		{"no static user", Middleware(WithRemoteAuth(false)), "", http.StatusUnauthorized, ""},
		{"skipped", Middleware(WithSkipper(func(echo.Context) bool { return true })), "", http.StatusOK, ""},
	}

	for _, scenario := range cases {
		t.Run(scenario.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)

			if scenario.token != "" {
				request.Header.Set(HeaderAuthToken, scenario.token)
			}

			recorder := httptest.NewRecorder()
			err := scenario.middleware(handler)(server.NewContext(request, recorder))

			if scenario.status == http.StatusOK {
				require.NoError(t, err)
				require.Equal(t, scenario.uid, recorder.Body.String())

				return
			}

			var response *rest.Error

			require.ErrorAs(t, err, &response)
			require.Equal(t, scenario.status, response.Code)
		})
	}
}
