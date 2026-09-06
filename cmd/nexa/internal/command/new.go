package command

import (
	"errors"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
	"nexis.run/nexa/cmd/nexa/internal/gen"
)

func (app *application) newCommand() *cobra.Command {
	settings := &writeSettings{}
	command := &cobra.Command{Use: "new", Short: "批量生成 DAO 和 Echo 上下文"}
	settings.bind(command, command.PersistentFlags(), true)

	var withDI, all bool

	daoCommand := &cobra.Command{
		Use:     "dao [NAME...]",
		Short:   "生成 DAO 并创建或更新依赖注入文件",
		Example: examples("nexa new dao User Order", "nexa new dao --all", "nexa new dao User --dry-run", "nexa new dao User --di=false"),
		Args: func(command *cobra.Command, names []string) error {
			if !all {
				return exportedIdentifierArgs(command, names)
			}

			if len(names) > 0 {
				return errors.New("--all 不能与名称同时使用")
			}

			return nil
		},
		ValidArgsFunction: app.completeSchemas,
		RunE: func(command *cobra.Command, names []string) (err error) {
			var generator *gen.Gen

			generator, err = app.generator(command)
			if err != nil {
				return
			}

			// --all 以当前 schema 列表作为名称
			if all {
				names, err = generator.Config.SchemaNames()
				if err != nil {
					return
				}

				if len(names) == 0 {
					err = errors.New("未找到 Ent schema，可运行 nexa ent new NAME")
					return
				}
			}

			var files []fileplan.File

			files, err = generator.PlanDAO(names, settings.force, withDI)
			if err != nil {
				return
			}

			err = app.applyFiles(command, settings, files)

			return
		},
	}
	daoCommand.Flags().BoolVarP(&withDI, "di", "d", true, "创建或更新依赖注入文件")
	daoCommand.Flags().BoolVar(&all, "all", false, "为全部 Ent schema 生成 DAO，不能与名称同时使用")

	echoCommand := &cobra.Command{
		Use:     "echoctx NAME [NAME...]",
		Short:   "生成可直接使用的 Echo 上下文和中间件",
		Example: examples("nexa new echoctx Rider Operator", "nexa new echoctx Rider --check"),
		Args:    exportedIdentifierArgs,
		RunE: func(command *cobra.Command, names []string) (err error) {
			var generator *gen.Gen

			generator, err = app.generator(command)
			if err != nil {
				return
			}

			var files []fileplan.File

			files, err = generator.PlanEchoContext(names, settings.force)
			if err != nil {
				return
			}

			err = app.applyFiles(command, settings, files)

			return
		},
	}
	command.AddCommand(daoCommand, echoCommand)

	return command
}

func (app *application) generator(command *cobra.Command) (generator *gen.Gen, err error) {
	var config *base.Config

	config, err = app.loadConfig(command)
	if err != nil {
		return
	}

	generator, err = gen.New(config)

	return
}
