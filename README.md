# Nexa

Nexa 是一个面向 **NEXA 框架** 的实用工具（CLI），用于辅助项目初始化、配置管理与代码生成等工作。

> CLI 基于 [cobra](https://github.com/spf13/cobra) 实现，支持通过 `-c/--config` 指定配置文件（默认 `.nexa.yaml`）。

## Install

### Option 1: install script

```bash
curl -fsSL https://nexis.run/nexa | bash
```

### Option 2: go install

```bash
go install nexis.run/nexa/cmd/nexa@latest
```

### Option 3: build from source

```bash
git clone https://github.com/nexis-run/nexa.git
cd nexa
go build -o nexa ./cmd/nexa
./nexa --help
```

## Usage

```bash
# 查看帮助
nexa --help

# 查看版本
nexa version

# 指定配置文件（默认 .nexa.yaml）
nexa -c .nexa.yaml <command>
```

## Configuration

- 默认配置文件名：`.nexa.yaml`
- 可以通过 `-c/--config` 指定其它配置文件路径
- 通过 `config init` 可以生成默认配置文件

## Commands

### config

初始化/管理配置。

```bash
# 在当前配置路径生成默认配置文件（若已存在会失败）
nexa config init
```

### new

生成代码模板（支持覆盖输出文件）。

```bash
# 覆盖已存在文件
nexa new --force <subcommand>
# 或
nexa new -f <subcommand>
```

#### new dao

新建数据访问对象（DAO）模板。

- 参数 `names` 必须以大写字母开头（例如 `User`、`OrderItem`）

```bash
nexa new dao User
nexa new dao User --force
```

#### new echoctx

已注册该子命令，但 README 暂未补充详细说明（以 `nexa new --help` 为准）。

### ent

ent 相关命令。

#### ent new

新建 ent schema（名称必须以大写字母开头）。

```bash
nexa ent new Example
```

#### ent generate

根据 ent schema 生成代码。

```bash
nexa ent generate
```

## Development

### Commit message convention

参考：[Commit message 和 Change log 编写指南](https://www.ruanyifeng.com/blog/2016/01/commit_message_change_log.html)

建议格式：

```text
[<type>](<scope>) <subject> (#pr)

docs:          文档变动
fix:           bug 修复
feat:          新增功能
feat-wip:      开发中的功能
improvement:   原有功能优化
style:         代码风格调整
typo:          代码或文档勘误
refactor:      代码重构（不涉及功能变动）
performance:   性能优化
test:          单元测试添加或修复
chore:         构建工具修改
revert:        回滚
deps:          第三方依赖修改
community:     社区相关修改
```

## License

MIT License. See [LICENSE](LICENSE).
