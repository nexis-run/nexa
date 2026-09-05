// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-27, by liasica

package authz

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"gopkg.auroraride.com/rbac"
)

const (
	allowedPermission = "allowed_permission"
	existingProject   = "MONETA_MANAGE"
	testToken         = "test-token"
	testUID           = "test_uid"
)

type testServer struct {
	rbac.UnimplementedRBACServiceServer
}

func (*testServer) GetRestrictedUser(_ context.Context, request *rbac.GetRestrictedUserRequest) (*rbac.GetRestrictedUserResponse, error) {
	hasPermission := request.PermissionKey == allowedPermission
	hasUser := request.ProjectCode.String() == existingProject

	var user *rbac.User

	if hasUser {
		user = &rbac.User{
			Uid: uuid.New().String(),
		}
	}

	return &rbac.GetRestrictedUserResponse{
		HasPermission: hasPermission,
		UserInfo:      user,
	}, nil
}

func (*testServer) GetUser(_ context.Context, request *rbac.GetUserRequest) (*rbac.GetUserResponse, error) {
	var user *rbac.User

	if request.Uid == testUID {
		user = &rbac.User{
			Uid: testUID,
		}
	}

	return &rbac.GetUserResponse{
		UserInfo: user,
	}, nil
}

func TestClient(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	rbac.RegisterRBACServiceServer(server, &testServer{})

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		_ = Close()
		server.Stop()
	})

	err = Setup(listener.Addr().String())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	t.Run("restricted user", func(t *testing.T) {
		var response *rbac.GetRestrictedUserResponse
		var err error

		response, err = GetRestrictedUser(
			ctx,
			testToken,
			existingProject,
			allowedPermission,
		)
		require.NoError(t, err)
		require.True(t, response.HasPermission)
		require.NotNil(t, response.UserInfo)

		response, err = GetRestrictedUser(
			ctx,
			testToken,
			existingProject,
			"forbidden_permission",
		)
		require.NoError(t, err)
		require.False(t, response.HasPermission)
		require.NotNil(t, response.UserInfo)

		response, err = GetRestrictedUser(
			ctx,
			testToken,
			"non_existing_project",
			allowedPermission,
		)
		require.NoError(t, err)
		require.True(t, response.HasPermission)
		require.Nil(t, response.UserInfo)
	})

	t.Run("user", func(t *testing.T) {
		user, err := GetUser(ctx, testUID)
		require.NoError(t, err)
		require.NotNil(t, user)
		require.Equal(t, testUID, user.Uid)
	})

	t.Run("metadata", func(t *testing.T) {
		base := metadata.NewOutgoingContext(ctx, metadata.Pairs(
			"x-request-id", "request-id",
			"authorization", "Bearer stale-token",
		))
		withAuth := GetRBACContext(base, testToken)
		outgoing, ok := metadata.FromOutgoingContext(withAuth)
		require.True(t, ok)
		require.Equal(t, []string{"request-id"}, outgoing.Get("x-request-id"))
		require.Equal(t, []string{"Bearer " + testToken}, outgoing.Get("authorization"))
	})
}

func TestGetUserWithoutSetup(t *testing.T) {
	_ = Close()

	_, err := GetUser(context.Background(), testUID)
	require.ErrorIs(t, err, ErrNotInitialized)
}
