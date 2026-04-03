// Copyright (C) micros. 2025-present.
//
// Created at 2025-01-06, by liasica

package kit

type Environment string

const (
	Production  Environment = "production"  // 生产环境
	Staging     Environment = "staging"     // 预发布环境, 模拟生产环境进行测试, 用作测试环境
	Development Environment = "development" // 开发环境, 本地开发使用
)

func (e Environment) IsValid() bool {
	switch e {
	case Production, Staging, Development:
		return true
	}

	return false
}

func (e Environment) IsProduction() bool {
	return e == Production
}
