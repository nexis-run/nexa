# NEXA CLI 快速入门指南

## 安装

### 自动安装

```bash
curl -fsSL https://nexis.run/nexa | bash
```

安装脚本会从 [GitHub Releases](https://github.com/nexis-run/nexa/releases) 下载当前操作系统和架构对应的可执行文件，并安装到 `$(go env GOPATH)/bin`。

### 使用 Go 安装

```bash
go install nexis.run/nexa/cmd/nexa@latest
```

### 验证安装

```bash
nexa --version
```

## 本地开发

### 前置要求

- Go 1.25.3+
- Git
- Make

### 克隆仓库

```bash
git clone https://github.com/nexis-run/nexa.git
cd nexa
```

### 构建

```bash
VERSION=0.1.0 make build-darwin-arm64
VERSION=0.1.0 make build-darwin-amd64
VERSION=0.1.0 make build-linux-amd64
VERSION=0.1.0 make build-linux-arm64
VERSION=0.1.0 make build-windows-amd64
```

构建所有支持的平台：

```bash
VERSION=0.1.0 make all
```

构建产物位于 `bin/` 目录。

## 发布

向 `master` 分支推送 CLI、构建配置或发布工作流变更后，GitHub Actions 会构建以下文件并创建 Release：

- `nexa-linux-amd64`
- `nexa-linux-arm64`
- `nexa-darwin-amd64`
- `nexa-darwin-arm64`
- `nexa-windows-amd64.exe`

Release 标签格式为 `{major}.{minor}.{patch}.{hash}`，例如 `0.1.0.c39a3be`。也可在 GitHub Actions 页面手动运行发布工作流。

## 常用命令

```bash
nexa --help
nexa version
nexa config init
nexa new dao User
nexa ent new Example
nexa ent generate
```

## 获取帮助

- [Issues](https://github.com/nexis-run/nexa/issues)
- [项目文档](.)
