// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-27, by liasica

package authz

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gopkg.auroraride.com/rbac"

	"nexis.run/nexa/kit/micro"
)

var (
	testAddress       = ":18001"
	allowedPermission = "allowed_permission"
	existingProject   = "MONETA_MANAGE"
	testToken         = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJjaWQiOiIiLCJleHAiOjE3NjE3Mjk3MTYsImlzcyI6IuaegeWFieWHuuihjCIsInByb2plY3RDb2RlIjpbInByb21vdGlvbi1tYW5hZ2UiLCJyZXBlcnRvcnktbWFuYWdlIiwic3NvLW1hbmFnZSIsInNnai1tYW5hZ2UiLCJkYXNoYm9hcmQtbWFuYWdlIl0sInN1YiI6Imx4ZiIsInVpZCI6NTd9.VPMRiKTwlu8p9qndYAhyYMZTrk2-Wrv9PV3QfPCD8nM"
	testUID           = "test_uid"
)

type testServer struct {
	rbac.UnimplementedRBACServiceServer
}

func (*testServer) GetRestrictedUser(_ context.Context, req *rbac.GetRestrictedUserRequest) (*rbac.GetRestrictedUserResponse, error) {
	hasPermission := req.PermissionKey == allowedPermission
	hasUser := req.ProjectCode.String() == existingProject

	var user *rbac.User

	if hasUser {
		user = &rbac.User{
			Uid: uuid.New().String(),
		}
	}

	res := &rbac.GetRestrictedUserResponse{
		HasPermission:       hasPermission,
		UserInfo:            user,
		DataPermissionRules: nil,
	}

	return res, nil
}

func (*testServer) GetUser(_ context.Context, req *rbac.GetUserRequest) (*rbac.GetUserResponse, error) {
	var user *rbac.User

	if req.Uid == testUID {
		user = &rbac.User{
			Uid: testUID,
		}
	}

	return &rbac.GetUserResponse{
		UserInfo: user,
	}, nil
}

func TestServer(_ *testing.T) {
	l, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(l)

	micro.Run("test-app", testAddress, func(s *grpc.Server) {
		rbac.RegisterRBACServiceServer(s, &testServer{})
	}, micro.LoggingMiddlewareServerOption())

	select {}
}

func TestGetRestrictedUser(t *testing.T) {
	Setup(testAddress)

	ctx := context.Background()

	res, err := GetRestrictedUser(ctx, testToken, existingProject, allowedPermission)
	require.NoError(t, err)
	require.True(t, res.HasPermission)
	require.NotNil(t, res.UserInfo)

	res, err = GetRestrictedUser(ctx, testToken, existingProject, "forbidden_permission")
	require.NoError(t, err)
	require.False(t, res.HasPermission)
	require.NotNil(t, res.UserInfo)

	res, err = GetRestrictedUser(ctx, testToken, "non_existing_project", allowedPermission)
	require.NoError(t, err)
	require.True(t, res.HasPermission)
	require.Nil(t, res.UserInfo)
}

func TestGetUser(t *testing.T) {
	Setup(testAddress)

	ctx := context.Background()

	res, err := GetUser(ctx, testUID)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, testUID, res.Uid)
}
