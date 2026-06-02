// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-25, by liasica

package authz

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"gopkg.auroraride.com/rbac"
)

var (
	instance  rbac.RBACServiceClient
	conn      *grpc.ClientConn
	setupOnce sync.Once
)

var _ = Setup

// SetupOption 初始化选项
type SetupOption func(*setupConfig)

type setupConfig struct {
	creds credentials.TransportCredentials
}

// WithTransportCredentials 设置 gRPC 传输凭证（TLS）
func WithTransportCredentials(creds credentials.TransportCredentials) SetupOption {
	return func(c *setupConfig) {
		c.creds = creds
	}
}

// Setup 初始化 rbac gRPC 客户端
// 如果初始化失败, 会直接抛出致命错误
func Setup(address string, opts ...SetupOption) {
	setupOnce.Do(func() {
		cfg := &setupConfig{
			creds: insecure.NewCredentials(),
		}

		for _, opt := range opts {
			opt(cfg)
		}

		var err error

		conn, err = grpc.NewClient(address, grpc.WithTransportCredentials(cfg.creds))
		if err != nil {
			zap.L().Fatal("rbac rpc连接失败", zap.Error(err))
			return
		}

		instance = rbac.NewRBACServiceClient(conn)
	})
}

// Close 关闭 gRPC 连接
func Close() error {
	if conn != nil {
		return conn.Close()
	}

	return nil
}

// GetRBACContext 取得权限服务上下文并添加认证信息
func GetRBACContext(ctx context.Context, token string) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"Authorization": "Bearer " + token}))
}

// GetRestrictedUser 检查权限
func GetRestrictedUser(ctx context.Context, token string, projectCode string, permissionKey string, opts ...Option) (*rbac.GetRestrictedUserResponse, error) {
	o := defaultOption

	for _, opt := range opts {
		opt.apply(&o)
	}

	res, err := instance.GetRestrictedUser(GetRBACContext(ctx, token), &rbac.GetRestrictedUserRequest{
		PermissionKey: permissionKey,
		ProjectCode:   rbac.GetProjectCode(projectCode),
	})

	if o.errorHandler != nil {
		err = o.errorHandler(err)
	}

	return res, err
}

// GetUser 获取用户信息
func GetUser(ctx context.Context, uid string, opts ...Option) (*rbac.User, error) {
	o := defaultOption

	for _, opt := range opts {
		opt.apply(&o)
	}

	res, err := instance.GetUser(ctx, &rbac.GetUserRequest{
		Uid: uid,
	})

	if o.errorHandler != nil {
		err = o.errorHandler(err)
	}

	if err != nil {
		return nil, err
	}

	return res.UserInfo, nil
}
