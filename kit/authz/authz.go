// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-25, by liasica

package authz

import (
	"context"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"gopkg.auroraride.com/rbac"
)

var defaultClient atomic.Pointer[Client]

type Client struct {
	instance rbac.RBACServiceClient
	conn     *grpc.ClientConn

	closeOnce sync.Once
	closeErr  error
}

// SetupOption 初始化选项
type SetupOption func(*setupConfig)

type setupConfig struct {
	creds credentials.TransportCredentials
}

// WithTransportCredentials 设置 gRPC 传输凭证（TLS）
func WithTransportCredentials(creds credentials.TransportCredentials) SetupOption {
	return func(config *setupConfig) {
		config.creds = creds
	}
}

// New 创建 rbac gRPC 客户端
func New(address string, opts ...SetupOption) (client *Client, err error) {
	config := &setupConfig{
		creds: insecure.NewCredentials(),
	}

	for _, opt := range opts {
		opt(config)
	}

	var conn *grpc.ClientConn

	conn, err = grpc.NewClient(address, grpc.WithTransportCredentials(config.creds))
	if err != nil {
		return
	}

	client = &Client{
		instance: rbac.NewRBACServiceClient(conn),
		conn:     conn,
	}

	return
}

// Setup 替换默认 rbac gRPC 客户端
func Setup(address string, opts ...SetupOption) (err error) {
	var client *Client

	client, err = New(address, opts...)
	if err != nil {
		return
	}

	previous := defaultClient.Swap(client)
	if previous != nil {
		err = previous.Close()
	}

	return
}

// Close 关闭默认 gRPC 客户端
func Close() (err error) {
	client := defaultClient.Swap(nil)
	if client == nil {
		return
	}

	err = client.Close()

	return
}

// Close 关闭 gRPC 客户端
func (client *Client) Close() (err error) {
	client.closeOnce.Do(func() {
		client.closeErr = client.conn.Close()
	})

	err = client.closeErr

	return
}

// GetRBACContext 取得权限服务上下文并添加认证信息
func GetRBACContext(ctx context.Context, token string) context.Context {
	outgoing, _ := metadata.FromOutgoingContext(ctx)
	outgoing = outgoing.Copy()
	outgoing.Set("authorization", "Bearer "+token)

	return metadata.NewOutgoingContext(ctx, outgoing)
}

// GetRestrictedUser 检查权限
func GetRestrictedUser(
	ctx context.Context,
	token string,
	projectCode string,
	permissionKey string,
	opts ...Option,
) (response *rbac.GetRestrictedUserResponse, err error) {
	client := defaultClient.Load()
	if client == nil {
		err = ErrNotInitialized
		return
	}

	response, err = client.GetRestrictedUser(
		ctx,
		token,
		projectCode,
		permissionKey,
		opts...,
	)

	return
}

// GetRestrictedUser 检查权限
func (client *Client) GetRestrictedUser(
	ctx context.Context,
	token string,
	projectCode string,
	permissionKey string,
	opts ...Option,
) (response *rbac.GetRestrictedUserResponse, err error) {
	settings := defaultOption

	for _, opt := range opts {
		opt.apply(&settings)
	}

	response, err = client.instance.GetRestrictedUser(
		GetRBACContext(ctx, token),
		&rbac.GetRestrictedUserRequest{
			PermissionKey: permissionKey,
			ProjectCode:   rbac.GetProjectCode(projectCode),
		},
	)

	if settings.errorHandler != nil {
		err = settings.errorHandler(err)
	}

	if err == nil && response == nil {
		err = ErrEmptyResponse
	}

	return
}

// GetUser 获取用户信息
func GetUser(ctx context.Context, uid string, opts ...Option) (user *rbac.User, err error) {
	client := defaultClient.Load()
	if client == nil {
		err = ErrNotInitialized
		return
	}

	user, err = client.GetUser(ctx, uid, opts...)

	return
}

// GetUser 获取用户信息
func (client *Client) GetUser(ctx context.Context, uid string, opts ...Option) (user *rbac.User, err error) {
	settings := defaultOption

	for _, opt := range opts {
		opt.apply(&settings)
	}

	var response *rbac.GetUserResponse
	response, err = client.instance.GetUser(ctx, &rbac.GetUserRequest{
		Uid: uid,
	})

	if settings.errorHandler != nil {
		err = settings.errorHandler(err)
	}

	if err != nil {
		return
	}

	if response == nil {
		err = ErrEmptyResponse
		return
	}

	user = response.UserInfo

	return
}
