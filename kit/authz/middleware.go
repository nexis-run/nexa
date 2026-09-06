// Copyright (C) nexa. 2026-present.
//
// Created at 2026-09-06, by liasica

package authz

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gopkg.auroraride.com/rbac"

	"nexis.run/nexa/kit/rest"
)

const (
	// ContextKeyUser 通过权限校验的用户在 echo 上下文中的键
	ContextKeyUser = "_user"

	HeaderAuthToken     = "X-Auth-Token"     // 权限校验 token
	HeaderProjectCode   = "X-Project-Code"   // 项目编码
	HeaderPermissionKey = "X-Permission-Key" // 权限键
)

// MiddlewareConfig 权限控制中间件配置
type MiddlewareConfig struct {
	EnableRemoteAuth bool               // 是否启用远程权限验证
	StaticUser       *rbac.User         // 不使用远程验证时的静态用户
	Skipper          middleware.Skipper // 跳过函数
	ProjectCode      string             // 项目代码，非空值优先于请求头
	PermissionKey    string             // 路由所需权限，非空值优先于请求头
	Client           *Client            // 权限客户端，空值使用默认客户端
}

type MiddlewareOption func(*MiddlewareConfig)

// WithPermissionKey 绑定路由所需权限，非空值优先于请求头
func WithPermissionKey(key string) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.PermissionKey = key
	}
}

// WithClient 设置独立权限客户端
func WithClient(client *Client) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.Client = client
	}
}

// WithRemoteAuth 设置是否启用远程权限验证
func WithRemoteAuth(enable bool) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.EnableRemoteAuth = enable
	}
}

// WithStaticUser 设置静态用户信息
func WithStaticUser(user *rbac.User) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.StaticUser = user
	}
}

// WithSkipper 设置跳过函数
func WithSkipper(skipper middleware.Skipper) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.Skipper = skipper
	}
}

// WithProjectCode 设置项目代码
func WithProjectCode(projectCode string) MiddlewareOption {
	return func(cfg *MiddlewareConfig) {
		cfg.ProjectCode = projectCode
	}
}

// Middleware 权限控制中间件，校验通过的用户通过 UserFromContext 读取
func Middleware(opts ...MiddlewareOption) echo.MiddlewareFunc {
	cfg := &MiddlewareConfig{EnableRemoteAuth: true}

	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if cfg.Skipper != nil && cfg.Skipper(c) {
				return next(c)
			}

			var (
				user          *rbac.User
				hasPermission bool
			)

			// 远程验证要求携带 token，由权限服务返回用户与权限结果
			if cfg.EnableRemoteAuth {
				token := c.Request().Header.Get(HeaderAuthToken)
				if token == "" {
					return rest.WrapError(http.StatusUnauthorized, ErrUnauthorized)
				}

				// 权限键与项目代码以配置值优先，未配置时读取请求头
				permissionKey := cfg.PermissionKey
				if permissionKey == "" {
					permissionKey = c.Request().Header.Get(HeaderPermissionKey)
				}

				projectCode := cfg.ProjectCode
				if projectCode == "" {
					projectCode = c.Request().Header.Get(HeaderProjectCode)
				}

				getRestrictedUser := GetRestrictedUser

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
					return ErrEmptyResponse
				}

				user = authed.UserInfo
				hasPermission = authed.HasPermission
			} else if cfg.StaticUser != nil {
				// 静态用户仅用于开发与测试环境，视为拥有全部权限
				user = cfg.StaticUser
				hasPermission = true
			}

			if user == nil {
				return rest.WrapError(http.StatusUnauthorized, ErrUnauthorized)
			}

			// 用户信息在权限判定前写入上下文，供后续错误处理与日志使用
			c.Set(ContextKeyUser, user)

			if !hasPermission {
				return rest.WrapError(http.StatusForbidden, ErrForbidden)
			}

			return next(c)
		}
	}
}

// UserFromContext 返回通过权限校验的用户，未校验时为 nil
func UserFromContext(c echo.Context) *rbac.User {
	user, _ := c.Get(ContextKeyUser).(*rbac.User)

	return user
}
