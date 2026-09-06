// Copyright (C) nexa. 2025-present.
//
// Created at 2025-10-25, by liasica

package rest

import (
	"net/http"

	"github.com/labstack/echo/v4"
	ew "github.com/labstack/echo/v4/middleware"
	"gopkg.auroraride.com/rbac"

	"nexis.run/nexa/kit/authz"
)

// RBACMiddlewareConfig 权限控制中间件配置
type RBACMiddlewareConfig struct {
	EnableRemoteAuth bool          // 是否启用远程权限验证
	StaticUser       *rbac.User    // 静态用户信息（当不使用远程验证时）
	Skipper          ew.Skipper    // 跳过函数
	ProjectCode      string        // 项目代码
	PermissionKey    string        // 服务端绑定的权限键
	Client           *authz.Client // 权限客户端，空值使用默认客户端
}

type RBACMiddlewareOption func(*RBACMiddlewareConfig)

// WithRBACPermissionKey 绑定路由所需权限，非空值优先于请求头
func WithRBACPermissionKey(key string) RBACMiddlewareOption {
	return func(cfg *RBACMiddlewareConfig) {
		cfg.PermissionKey = key
	}
}

// WithRBACClient 设置独立权限客户端
func WithRBACClient(client *authz.Client) RBACMiddlewareOption {
	return func(cfg *RBACMiddlewareConfig) {
		cfg.Client = client
	}
}

// WithRBACRemoteAuth 设置是否启用远程权限验证
func WithRBACRemoteAuth(enable bool) RBACMiddlewareOption {
	return func(cfg *RBACMiddlewareConfig) {
		cfg.EnableRemoteAuth = enable
	}
}

// WithRBACStaticUser 设置静态用户信息
func WithRBACStaticUser(user *rbac.User) RBACMiddlewareOption {
	return func(cfg *RBACMiddlewareConfig) {
		cfg.StaticUser = user
	}
}

// WithRBACSkipper 设置跳过函数
func WithRBACSkipper(skipper ew.Skipper) RBACMiddlewareOption {
	return func(cfg *RBACMiddlewareConfig) {
		cfg.Skipper = skipper
	}
}

// WithRBACProjectCode 设置项目代码
func WithRBACProjectCode(projectCode string) RBACMiddlewareOption {
	return func(cfg *RBACMiddlewareConfig) {
		cfg.ProjectCode = projectCode
	}
}

// RBACMiddleware 权限控制中间件
func RBACMiddleware(opts ...RBACMiddlewareOption) echo.MiddlewareFunc {
	// 在构造阶段解析配置
	cfg := &RBACMiddlewareConfig{
		EnableRemoteAuth: true,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c = ensureContext(c)
			ctx := GetContext(c)

			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			token := c.Request().Header.Get(HeaderAuthToken)

			// 权限键与项目代码以配置值优先，未配置时读取请求头
			permissionKey := cfg.PermissionKey
			if permissionKey == "" {
				permissionKey = c.Request().Header.Get(HeaderPermissionKey)
			}

			projectCode := cfg.ProjectCode
			if projectCode == "" {
				projectCode = c.Request().Header.Get(HeaderProjectCode)
			}

			var (
				user          *rbac.User
				hasPermission bool
			)

			// 远程验证要求携带 token，由权限服务返回用户与权限结果
			if cfg.EnableRemoteAuth {
				if token == "" {
					return WrapError(http.StatusUnauthorized, authz.ErrUnauthorized)
				}

				getRestrictedUser := authz.GetRestrictedUser

				if cfg.Client != nil {
					getRestrictedUser = cfg.Client.GetRestrictedUser
				}

				authed, err := getRestrictedUser(
					c.Request().Context(),
					token,
					projectCode,
					permissionKey,
				)
				if err != nil {
					return err
				}

				if authed == nil {
					return authz.ErrEmptyResponse
				}

				user = authed.UserInfo
				hasPermission = authed.HasPermission
			}

			// 静态用户仅用于开发与测试环境，视为拥有全部权限
			if !cfg.EnableRemoteAuth && cfg.StaticUser != nil {
				user = cfg.StaticUser
				hasPermission = true
			}

			if user == nil {
				return WrapError(http.StatusUnauthorized, authz.ErrUnauthorized)
			}

			// 用户信息在权限判定前写入上下文，供后续错误处理与日志使用
			ctx.User = user
			c.Set(ContextKeyUser, user)

			if !hasPermission {
				return WrapError(http.StatusForbidden, authz.ErrForbidden)
			}

			return next(c)
		}
	}
}
