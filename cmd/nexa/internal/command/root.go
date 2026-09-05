package command

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/nexa/internal/fileplan"
)

type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit,omitempty"`
	BuildTime string `json:"build_time,omitempty"`
	GoVersion string `json:"go_version"`
	Modified  bool   `json:"modified"`
}

func (info VersionInfo) String() string {
	commit := info.Commit
	if len(commit) > 12 {
		commit = commit[:12]
	}

	if info.Modified && commit != "" {
		commit += "-dirty"
	}

	var details []string

	if commit != "" {
		details = append(details, "commit "+commit)
	}

	if info.BuildTime != "" {
		details = append(details, "built at "+info.BuildTime)
	}

	if len(details) == 0 {
		return info.Version
	}

	return info.Version + " (" + strings.Join(details, ", ") + ")"
}

type application struct {
	configPath string
	json       bool
	version    VersionInfo
}

type ExitError struct {
	Code     int
	Reported bool
	Err      error
}

func (err *ExitError) Error() string { return err.Err.Error() }

func (err *ExitError) Unwrap() error { return err.Err }

func NewRoot(info VersionInfo) *cobra.Command {
	app := &application{version: info}
	command := &cobra.Command{
		Use:               "nexa",
		Short:             "配置、检查和生成 Go 项目代码",
		Version:           info.String(),
		SilenceErrors:     true,
		SilenceUsage:      true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	command.SetOut(os.Stdout)
	command.SetErr(os.Stderr)
	command.SetVersionTemplate("{{.Version}}\n")
	command.PersistentFlags().StringVarP(&app.configPath, "config", "c", "", "指定配置文件，默认在当前模块内查找 .nexa.yaml")
	command.PersistentFlags().BoolVar(&app.json, "json", false, "输出 JSON 结果（Shell 补全除外）")
	command.AddCommand(
		app.configCommand(),
		app.newCommand(),
		app.entCommand(),
		app.doctorCommand(),
		app.versionCommand(),
		completionCommand(),
	)
	initializeFlags(command)

	return command
}

func initializeFlags(command *cobra.Command) {
	command.InitDefaultHelpFlag()
	command.InitDefaultVersionFlag()

	for _, child := range command.Commands() {
		initializeFlags(child)
	}
}

func (app *application) versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "输出版本和构建信息",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) (err error) {
			if app.json {
				err = writeJSON(command.OutOrStdout(), app.version)
				return
			}

			command.Println(app.version.String())

			return
		},
	}
}

func completionCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "输出 Shell 补全脚本",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(command *cobra.Command, args []string) error {
			writer := command.OutOrStdout()
			root := command.Root()

			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(writer, true)
			case "zsh":
				return root.GenZshCompletion(writer)
			case "fish":
				return root.GenFishCompletion(writer, true)
			default:
				return root.GenPowerShellCompletionWithDesc(writer)
			}
		},
	}
}

type writeSettings struct {
	force  bool
	dryRun bool
	check  bool
}

func (settings *writeSettings) bind(command *cobra.Command, force bool) {
	if force {
		command.Flags().BoolVarP(&settings.force, "force", "f", false, "覆盖内容不同的已有文件")
	}

	command.Flags().BoolVar(&settings.dryRun, "dry-run", false, "列出文件变更，不应用变更")
	command.Flags().BoolVar(&settings.check, "check", false, "检查文件是否需要更新，有差异时以状态码 2 退出")
	command.MarkFlagsMutuallyExclusive("dry-run", "check")
}

func (settings *writeSettings) bindPersistent(command *cobra.Command, force bool) {
	if force {
		command.PersistentFlags().BoolVarP(&settings.force, "force", "f", false, "覆盖内容不同的已有文件")
	}

	command.PersistentFlags().BoolVar(&settings.dryRun, "dry-run", false, "列出文件变更，不应用变更")
	command.PersistentFlags().BoolVar(&settings.check, "check", false, "检查文件是否需要更新，有差异时以状态码 2 退出")
	command.MarkFlagsMutuallyExclusive("dry-run", "check")
}

func (app *application) applyFiles(command *cobra.Command, settings *writeSettings, files []fileplan.File) (err error) {
	if settings.check || settings.dryRun {
		for index := range files {
			files[index].Overwrite = true
		}
	}

	var plan *fileplan.Plan

	plan, err = fileplan.New(files...)
	if err != nil {
		return
	}

	if !settings.check && !settings.dryRun {
		err = plan.Apply(command.Context())
		if err != nil {
			return
		}
	}

	changes := plan.Changes()

	result := struct {
		Changes []fileplan.Change `json:"changes"`
		Applied bool              `json:"applied"`
		Current bool              `json:"current"`
	}{changes, !settings.check && !settings.dryRun, len(changes) == 0}
	if app.json {
		err = writeJSON(command.OutOrStdout(), result)
	} else if len(changes) == 0 {
		command.Println("文件内容已一致。")
	} else {
		for _, change := range changes {
			command.Printf("%s %s\n", change.Action, change.Path)
		}

		if settings.check || settings.dryRun {
			command.Printf("共 %d 项变更，未写入。\n", len(changes))
		} else {
			command.Printf("已应用 %d 项变更。\n", len(changes))
		}
	}

	if err == nil && settings.check && len(changes) > 0 {
		err = &ExitError{Code: 2, Reported: true, Err: errors.New("生成文件需要更新")}
	}

	return
}

func writeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(value)
}
