# CLI 快速入门

## 安装

```bash
go install nexis.run/nexa/cmd/nexa@latest
nexa version
```

预编译安装入口：

```bash
curl -fsSL https://nexis.run/nexa | bash
```

安装器只接受带有 SHA-256 校验文件的稳定 Release。目录优先级为 `NEXA_INSTALL_DIR`、Go 的 `GOBIN`、`GOPATH/bin`、`$HOME/.local/bin`。安装目录需要加入 `PATH`。

可通过 `--version vX.Y.Z` 指定已发布版本，`--force` 允许重新安装、降级或替换无法识别版本的程序。

## 项目使用

在能访问 Nexa 及其依赖的 Go 模块中运行：

```bash
nexa --help
nexa config init
nexa ent new User Order
go mod tidy
nexa ent generate
nexa new dao User Order
nexa new echoctx Rider
go mod tidy
go test ./...
nexa doctor
```

`ent new` 创建 schema 文件，`ent generate` 生成 Ent 客户端和扩展代码，`new dao` 根据实体生成 DAO，`new echoctx` 创建 Echo Context。名称支持一次传入多个，必须是大写字母开头的 Go 标识符，且不能生成 `_test.go` 或带有系统、架构后缀的文件，例如 `User_Test`、`Rider_Linux`。

DAO 和 Echo Context 使用现有 Go 包声明，目录名可以与包名不同。生成前检查整批输出与其他源文件中的类型、函数和变量声明冲突；`--force` 只允许覆盖目标文件。schema 列表、DAO 预检和包名解析都遵守当前平台的 Go 构建约束，不解析被排除的源文件。

默认 DAO 构造器采用显式注入：

```go
userDAO := dao.NewUser(client)
query := userDAO.Query()
entityClient := userDAO.Client()
transactionDAO := userDAO.Tx(tx)
```

`new dao` 默认维护配置中的 `Dao` 结构体与 `daoProviderSet`。DI 文件不存在时自动创建，已有 DI 保留自定义 provider 与别名，并补齐请求实体的 Wire 显式字段名单。已有实体的构造器由项目维护。应用使用 Wire 时，需要提供 `*ent.Client`；CLI 不创建数据库连接。只生成 DAO 文件时使用 `--di=false`。

使用 Ent 客户端时，在应用启动包空白导入生成包下的 `runtime`。DAO 已包含该导入：

```go
import _ "example.com/app/internal/infrastructure/ent/runtime"
```

## 命令与输出

| 命令 | 用途 | 写入选项 |
| --- | --- | --- |
| `config init` | 在当前目录或 `-c` 指定位置创建默认配置 | `--force`、`--dry-run`、`--check` |
| `config show` | 输出合并默认值后的 YAML 或 JSON | 无 |
| `config validate` | 校验配置字段、模块和输出路径 | 无 |
| `doctor` | 只读检查 Go、配置、Ent、schema 与 DI | 无 |
| `ent new NAME...` | 批量创建 Ent schema | `--force`、`--dry-run`、`--check` |
| `ent generate` | 生成 Ent 客户端与扩展，别名为 `ent gen` | `--dry-run`、`--check` |
| `ent list` | 静态列出 Ent schema | 无 |
| `new dao NAME...` | 批量创建 DAO 并维护 DI | `--di=false`、`--force`、`--dry-run`、`--check` |
| `new echoctx NAME...` | 批量创建 Echo Context | `--force`、`--dry-run`、`--check` |
| `version` | 输出版本和构建信息 | 无 |
| `completion SHELL` | 输出 Shell 补全脚本 | 无 |

全局 `--json` 用于脚本读取结果，不改变补全脚本格式。帮助、版本和补全脚本输出无需项目配置。诊断中的 `warning` 不导致命令失败，`error` 导致失败。

```bash
nexa doctor --json
nexa config show --json
nexa ent list --json
nexa new dao User Order --dry-run --json
nexa ent generate --check --json
```

文件变更结果包含 `changes`、`applied` 和 `current`。每项变更包含绝对 `path` 和 `action`（`create`、`update` 或 `delete`）。一般错误输出到标准错误；检查与诊断结果输出到标准输出。

| 状态码 | 含义 |
| --- | --- |
| `0` | 执行成功，或 `--check` 无差异 |
| `1` | 参数、配置、生成或写入失败 |
| `2` | `--check` 检测到差异 |
| `130` | 命令被取消 |

## 配置

默认配置文件名为 `.nexa.yaml`。CLI 在当前模块内向上查找配置，缺失时使用默认值。显式 `-c/--config` 只读取指定文件，文件缺失会报错。`config init` 的目标为当前目录或指定位置，不要求存在 `go.mod`。

```yaml
entPath: internal/infrastructure/ent
daoPath: internal/infrastructure/dao
echoctxPath: internal/app/rest/app
ormclient: ''
entFeatures: []
entTemplates: []
di:
  path: internal/di/di.go
  daoStructName: Dao
  daoProviderSetVar: daoProviderSet
```

输出路径以配置所属模块根目录为基准，不能越过模块边界或进入嵌套模块，输出目标不能是符号链接。配置拒绝未知字段、重复键、`null` 和多文档 YAML。`ormclient` 为空字符串时使用显式注入；非空时作为 DAO 构造器中的 Go 表达式，例如 `ent.Database`，对应表达式必须在项目中存在。

```bash
nexa -c config/nexa.yaml config init
nexa -c config/nexa.yaml config validate
nexa -c config/nexa.yaml ent generate
```

`entFeatures` 在默认特性上追加；默认包含 `sql/modifier`、`sql/upsert`、`privacy`、`entql`、`sql/execquery`、`intercept` 和 `schema/snapshot`。`entTemplates` 配置附加模板目录，路径相对模块根目录解析。

## 文件保护与生成边界

生成命令先渲染并预检整批目标，再暂存和应用变更。内容相同的文件保持不动；模板文件已有不同内容时，需要显式 `--force`。`--dry-run` 只列出变更，`--check` 只检查差异，两者不能同时使用。写入失败会尝试恢复本次已写入的文件；并发修改会报错，不保证多文件跨进程原子性。

`ent generate` 更新带有 Ent 生成标记的文件，删除不再输出的已生成文件，保留手写文件。与手写文件同名的输出会报错。直接编辑带生成标记的文件，其内容会受生成器管理。

Ent 生成在模块内临时目录运行，正常退出后清理临时目录。生成及其 `--dry-run`、`--check` 都会执行 schema 的 Go 代码，只对可信项目使用；仅 `ent list` 和 `doctor` 的 schema 检查是静态读取。Go 依赖需预先准备，CLI 不自动修改 `go.mod`、`go.sum`，缺依赖时先运行 `go mod tidy`。Ent 生成器与项目的 Ent 版本应保持一致，参见 [Ent 代码生成文档](https://entgo.io/docs/code-gen/)。

生成包下的 `generate.go` 支持 `go generate`，已有生成指令保持不变。自定义配置会写入对应的配置参数。

## Shell 补全

```bash
source <(nexa completion bash)
source <(nexa completion zsh)
nexa completion fish | source
nexa completion powershell | Out-String | Invoke-Expression
```

补全脚本可按所用 Shell 的启动配置持久加载；DAO 名称补全读取当前项目的 schema。

## 本地构建

需要 Go 1.25.3 或更高版本，以及 Git 和 Make。

```bash
git clone https://github.com/nexis-run/nexa.git
cd nexa
VERSION=0.1.1 make -j6 all
```

也可以只构建目标平台：

```bash
VERSION=0.1.1 make build-linux-amd64
```

产物位于 `bin/`。支持 Linux、macOS、Windows，各自提供 amd64 和 arm64 版本。构建统一使用 `CGO_ENABLED=0`。

## CI

CI 在 PR、`master` 分支推送和手动触发时执行 `go test -race ./cmd/nexa/...` 和 `go vet ./cmd/nexa/...`，检查公开 CLI。

## 发布

发布工作流由 `vX.Y.Z` 标签触发，标签必须对应稳定语义化版本。工作流验证公开 CLI，生成六个平台产物及 `checksums.txt`，先上传至草稿 Release，再公开发布。已公开版本不能被覆盖。

安装入口使用默认分支的安装脚本，因此上线安装脚本时必须保证已有与其匹配的稳定 Release 和校验文件。发布操作与安装入口上线由维护者执行。
