package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"nexis.run/nexa/cmd/nexa/internal/command"
	"nexis.run/nexa/cmd/nexa/internal/entgen"
)

var (
	Version   = "dev"
	BuildTime string
	Hash      string
)

func buildVersionInfo() command.VersionInfo {
	info := command.VersionInfo{
		Version:   Version,
		Commit:    Hash,
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
	}

	build, ok := debug.ReadBuildInfo()
	if !ok {
		return info
	}

	if (info.Version == "" || info.Version == "dev") && build.Main.Version != "" && build.Main.Version != "(devel)" {
		info.Version = strings.TrimPrefix(build.Main.Version, "v")
	}

	for _, setting := range build.Settings {
		switch setting.Key {
		case "vcs.revision":
			if info.Commit == "" {
				info.Commit = setting.Value
			}
		case "vcs.modified":
			info.Modified = setting.Value == "true"
		}
	}

	if info.Version == "" {
		info.Version = "dev"
	}

	return info
}

func newRootCommand() *cobra.Command {
	return command.NewRoot(buildVersionInfo())
}

func run(ctx context.Context) int {
	var err error
	jsonOutput := false

	if len(os.Args) > 1 && os.Args[1] == entgen.WorkerCommand {
		err = entgen.RunWorker(ctx, os.Stdin)
	} else {
		root := newRootCommand()
		err = root.ExecuteContext(ctx)

		// 命令匹配失败时仍按 Cobra 规则解析输出选项
		if err != nil {
			target, arguments, _ := root.Find(os.Args[1:])
			if target != nil && !target.Flags().Parsed() {
				_ = target.ParseFlags(arguments)
			}
		}

		jsonOutput, _ = root.PersistentFlags().GetBool("json")
	}

	if err == nil {
		return 0
	}

	code := 1
	var exit *command.ExitError

	if errors.As(err, &exit) {
		if exit.Reported {
			return exit.Code
		}

		code = exit.Code
	}

	if errors.Is(err, context.Canceled) || ctx.Err() != nil {
		code = 130
	}

	if jsonOutput {
		_ = json.NewEncoder(os.Stderr).Encode(map[string]any{"error": err.Error(), "exit_code": code})
	} else {
		_, _ = fmt.Fprintln(os.Stderr, err)
	}

	return code
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx)
	stop()

	os.Exit(code)
}
