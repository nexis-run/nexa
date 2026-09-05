package command

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/mod/modfile"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/entgen"
	"nexis.run/nexa/cmd/nexa/internal/parser"
)

type diagnostic struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (app *application) doctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "检查 Go 环境、配置、Ent 和依赖注入状态",
		Long:  "只读检查本地 Go 环境和项目生成条件，不下载依赖、不连接外部服务。",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) error {
			checks := []diagnostic{checkGo(command.Context())}

			config, err := app.loadConfig(command)
			if err != nil {
				checks = append(checks, diagnostic{"config", "error", err.Error()})
			} else {
				checks = append(checks, inspectProject(config)...)
			}

			return app.reportDiagnostics(command, checks)
		},
	}
}

func checkGo(ctx context.Context) diagnostic {
	path, err := exec.LookPath("go")
	if err != nil {
		return diagnostic{"go", "error", "未找到 Go，请安装 go.mod 指定的版本并加入 PATH"}
	}

	var cancel context.CancelFunc

	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, path, "version")

	command.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	var output []byte

	output, err = command.CombinedOutput()
	if err != nil {
		return diagnostic{"go", "error", strings.TrimSpace(string(output)) + "：" + err.Error()}
	}

	return diagnostic{"go", "ok", strings.TrimSpace(string(output))}
}

func inspectProject(config *base.Config) []diagnostic {
	checks := []diagnostic{{"config", "ok", config.GetConfigFilePath()}}
	if !config.ConfigExists() {
		checks[0].Status = "warning"
		checks[0].Message += "，正在使用默认值，可运行 nexa config init"
	}

	module, err := config.ResolveModule()
	if err != nil {
		return append(checks, diagnostic{"module", "error", err.Error()})
	}

	checks = append(checks, diagnostic{"module", "ok", module + "（" + config.RootDir + "）"})
	checks = append(checks, checkEntVersion(config.RootDir))

	var generator *entgen.EntGen

	generator, err = entgen.New(config)
	if err != nil {
		return append(checks, diagnostic{"schema", "error", err.Error()})
	}

	var names []string

	names, err = generator.SchemaNames()
	if err != nil {
		checks = append(checks, diagnostic{"schema", "error", err.Error()})
	} else if len(names) == 0 {
		checks = append(checks, diagnostic{"schema", "warning", "未找到 Ent schema，可运行 nexa ent new NAME"})
	} else {
		checks = append(checks, diagnostic{"schema", "ok", strings.Join(names, ", ")})
	}

	var diPath string

	diPath, err = config.GetDIPath()
	if err != nil {
		return append(checks, diagnostic{"di", "error", err.Error()})
	}

	var diContent []byte

	diContent, err = os.ReadFile(diPath)
	if errors.Is(err, os.ErrNotExist) {
		return append(checks, diagnostic{"di", "warning", "DI 文件尚未创建，nexa new dao 将创建此文件"})
	}

	if err != nil {
		return append(checks, diagnostic{"di", "error", err.Error()})
	}

	var daoImport, daoDirectory, daoPackage string

	daoImport, err = config.ResolvePackagePath(config.DaoPath)
	if err != nil {
		return append(checks, diagnostic{"di", "error", err.Error()})
	}

	daoDirectory, err = config.GetDaoPath()
	if err != nil {
		return append(checks, diagnostic{"di", "error", err.Error()})
	}

	daoPackage, err = base.ResolvePackageName(daoDirectory)
	if err != nil {
		return append(checks, diagnostic{"di", "error", err.Error()})
	}

	_, err = parser.ParseDaoProvider(parser.DaoProviderConfig{
		Path:         diPath,
		ImportPath:   daoImport,
		PackageName:  daoPackage,
		TypeName:     config.DI.DaoStructName,
		VariableName: config.DI.DaoProviderSetVar,
	}, diContent)
	if err != nil {
		checks = append(checks, diagnostic{"di", "warning", err.Error() + "；只生成 DAO 时可使用 --di=false"})
	} else {
		checks = append(checks, diagnostic{"di", "ok", diPath})
	}

	return checks
}

func checkEntVersion(root string) diagnostic {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return diagnostic{"ent", "error", err.Error()}
	}

	var module *modfile.File

	module, err = modfile.Parse("go.mod", content, nil)
	if err != nil {
		return diagnostic{"ent", "error", err.Error()}
	}

	for _, dependency := range module.Require {
		if dependency.Mod.Path == "entgo.io/ent" {
			for _, replacement := range module.Replace {
				if replacement.Old.Path == dependency.Mod.Path {
					return diagnostic{"ent", "warning", "项目替换了 Ent 依赖，请确认与 CLI 内置生成器兼容：" + replacement.New.Path + " " + replacement.New.Version}
				}
			}

			build, available := debug.ReadBuildInfo()
			if available {
				for _, compiled := range build.Deps {
					if compiled.Path == dependency.Mod.Path && compiled.Version != dependency.Mod.Version {
						return diagnostic{"ent", "warning", "项目 Ent 为 " + dependency.Mod.Version + "，CLI 生成器为 " + compiled.Version + "，请使用版本一致的 CLI"}
					}
				}
			}

			return diagnostic{"ent", "ok", dependency.Mod.Version}
		}
	}

	return diagnostic{"ent", "warning", "go.mod 未声明 entgo.io/ent，生成前请安装项目所需依赖"}
}

func (app *application) reportDiagnostics(command *cobra.Command, checks []diagnostic) (err error) {
	valid := true

	for _, check := range checks {
		valid = valid && check.Status != "error"
	}

	if app.json {
		err = writeJSON(command.OutOrStdout(), struct {
			Valid  bool         `json:"valid"`
			Checks []diagnostic `json:"checks"`
		}{valid, checks})
	} else {
		for _, check := range checks {
			command.Printf("[%s] %s：%s\n", check.Status, check.Name, check.Message)
		}
	}

	if err == nil && !valid {
		err = &ExitError{Code: 1, Reported: true, Err: errors.New("项目检查未通过")}
	}

	return
}
