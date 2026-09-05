package command

import (
	"strings"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/nexa/internal/base"
)

func examples(commands ...string) string {
	for index := range commands {
		commands[index] = "  " + commands[index]
	}

	return strings.Join(commands, "\n")
}

func exportedIdentifierArgs(_ *cobra.Command, names []string) error {
	return base.ValidateNames(names)
}
