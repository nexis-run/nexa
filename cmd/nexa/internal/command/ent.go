package command

import (
	"strings"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/entgen"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
)

func (app *application) entCommand() *cobra.Command {
	command := &cobra.Command{Use: "ent", Short: "管理 Ent schema 和生成代码"}
	command.AddCommand(app.entNewCommand(), app.entGenerateCommand(), app.entListCommand())

	return command
}

func (app *application) entNewCommand() *cobra.Command {
	settings := &writeSettings{}
	command := &cobra.Command{
		Use:     "new NAME [NAME...]",
		Short:   "批量创建 Ent schema",
		Args:    exportedIdentifierArgs,
		Example: examples("nexa ent new User Order", "nexa ent new User --dry-run"),
		RunE: func(command *cobra.Command, names []string) (err error) {
			var generator *entgen.EntGen

			generator, err = app.entGenerator(command)
			if err != nil {
				return
			}

			var files []fileplan.File

			files, err = generator.PlanNew(names, settings.force)
			if err != nil {
				return
			}

			err = app.applyFiles(command, settings, files)

			return
		},
	}
	settings.bind(command, true)

	return command
}

func (app *application) entGenerateCommand() *cobra.Command {
	settings := &writeSettings{}
	command := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen"},
		Short:   "生成 Ent 客户端和扩展代码",
		Args:    cobra.NoArgs,
		Example: examples("nexa ent generate", "nexa ent generate --check"),
		RunE: func(command *cobra.Command, _ []string) (err error) {
			var generator *entgen.EntGen

			generator, err = app.entGenerator(command)
			if err != nil {
				return
			}

			var files []fileplan.File

			files, err = generator.PlanGenerate(command.Context())
			if err != nil {
				return
			}

			err = app.applyFiles(command, settings, files)

			return
		},
	}
	settings.bind(command, false)

	return command
}

func (app *application) entListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出项目中的 Ent schema",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) (err error) {
			var generator *entgen.EntGen

			generator, err = app.entGenerator(command)
			if err != nil {
				return
			}

			var names []string

			names, err = generator.SchemaNames()
			if err != nil {
				return
			}

			if app.json {
				err = writeJSON(command.OutOrStdout(), names)
				return
			}

			for _, name := range names {
				command.Println(name)
			}

			return
		},
	}
}

func (app *application) entGenerator(command *cobra.Command) (generator *entgen.EntGen, err error) {
	var config *base.Config

	config, err = app.loadConfig(command)
	if err != nil {
		return
	}

	generator, err = entgen.New(config)

	return
}

func (app *application) completeSchemas(command *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
	generator, err := app.entGenerator(command)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string

	names, err = generator.SchemaNames()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var choices []string
	used := make(map[string]bool)

	for _, name := range args {
		used[name] = true
	}

	for _, name := range names {
		if strings.HasPrefix(name, prefix) && !used[name] {
			choices = append(choices, name)
		}
	}

	return choices, cobra.ShellCompDirectiveNoFileComp
}
