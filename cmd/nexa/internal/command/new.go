package command

import (
	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
	"nexis.run/nexa/cmd/nexa/internal/gen"
)

func (app *application) newCommand() *cobra.Command {
	settings := &writeSettings{}
	command := &cobra.Command{Use: "new", Short: "批量生成 DAO 和 Echo 上下文"}
	settings.bindPersistent(command, true)

	var withDI bool

	daoCommand := &cobra.Command{
		Use:               "dao NAME [NAME...]",
		Short:             "生成 DAO 并创建或更新依赖注入文件",
		Example:           examples("nexa new dao User Order", "nexa new dao User --dry-run", "nexa new dao User --di=false"),
		Args:              exportedIdentifierArgs,
		ValidArgsFunction: app.completeSchemas,
		RunE: func(command *cobra.Command, names []string) (err error) {
			var config *base.Config

			config, err = app.loadConfig(command)
			if err != nil {
				return
			}

			var generator *gen.Gen

			generator, err = gen.New(config)
			if err != nil {
				return
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

	echoCommand := &cobra.Command{
		Use:     "echoctx NAME [NAME...]",
		Short:   "生成可直接使用的 Echo 上下文和中间件",
		Example: examples("nexa new echoctx Rider Operator", "nexa new echoctx Rider --check"),
		Args:    exportedIdentifierArgs,
		RunE: func(command *cobra.Command, names []string) (err error) {
			var config *base.Config

			config, err = app.loadConfig(command)
			if err != nil {
				return
			}

			var generator *gen.Gen

			generator, err = gen.New(config)
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
