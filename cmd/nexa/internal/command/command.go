package command

import (
	"fmt"
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

func exportedIdentifierArgs(_ *cobra.Command, names []string) (err error) {
	if len(names) == 0 {
		err = base.ErrNameRequired
		return
	}

	seen := make(map[string]bool)

	for _, name := range names {
		if !base.StringIsExportedIdentifier(name) {
			err = fmt.Errorf("%w：%s", base.ErrInvalidExportedName, name)
			return
		}

		key := strings.ToLower(name)
		if seen[key] {
			err = fmt.Errorf("名称重复或生成文件名冲突：%s", name)
			return
		}

		seen[key] = true
	}

	return
}
