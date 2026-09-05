package command

import (
	"errors"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/nexa/internal/base"
	"nexis.run/nexa/cmd/nexa/internal/fileplan"
)

func (app *application) configCommand() *cobra.Command {
	command := &cobra.Command{Use: "config", Short: "初始化、查看和校验项目配置"}
	command.AddCommand(app.configInitCommand(), app.configShowCommand(), app.configValidateCommand())

	return command
}

func (app *application) configInitCommand() *cobra.Command {
	settings := &writeSettings{}
	command := &cobra.Command{
		Use:   "init",
		Short: "创建默认配置文件",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) (err error) {
			if command.Flags().Changed("config") && app.configPath == "" {
				return errors.New("显式配置路径不能为空")
			}

			var config *base.Config

			config, err = base.NewDefaultConfig(app.configPath)
			if err != nil {
				return
			}

			var content []byte

			content, err = config.MarshalYAMLBytes()
			if err != nil {
				return
			}

			err = app.applyFiles(command, settings, []fileplan.File{{
				Path:      config.GetConfigFilePath(),
				Content:   content,
				Overwrite: settings.force,
			}})

			return
		},
	}
	settings.bind(command, true)

	return command
}

func (app *application) configShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "输出合并默认值后的有效配置",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) (err error) {
			var config *base.Config

			config, err = app.loadConfig(command)
			if err != nil {
				return
			}

			if app.json {
				err = writeJSON(command.OutOrStdout(), config)
				return
			}

			var content []byte

			content, err = config.MarshalYAMLBytes()
			if err != nil {
				return
			}

			_, err = command.OutOrStdout().Write(content)

			return
		},
	}
}

func (app *application) configValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "校验配置字段和项目路径",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, _ []string) (err error) {
			var config *base.Config

			config, err = app.loadConfig(command)
			if err != nil {
				return
			}

			result := struct {
				Valid  bool   `json:"valid"`
				Path   string `json:"path"`
				Exists bool   `json:"exists"`
			}{true, config.GetConfigFilePath(), config.ConfigExists()}

			if app.json {
				err = writeJSON(command.OutOrStdout(), result)
				return
			}

			command.Printf("配置有效：%s\n", result.Path)

			if !result.Exists {
				command.Println("配置文件不存在，当前使用默认值。")
			}

			return
		},
	}
}

func (app *application) loadConfig(command *cobra.Command) (*base.Config, error) {
	explicit := command.Flags().Changed("config")

	return base.LoadConfig(base.LoadOptions{
		ConfigPath:   app.configPath,
		Explicit:     explicit,
		AllowMissing: !explicit,
	})
}
