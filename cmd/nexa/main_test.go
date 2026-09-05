package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorOutputFollowsParsedJSONFlag(t *testing.T) {
	for _, scenario := range []struct {
		name string
		args []string
		json bool
	}{
		{"unknown command", []string{"--json", "invalid"}, true},
		{"argument error", []string{"--json", "version", "extra"}, true},
		{"argument terminator", []string{"version", "--", "--json"}, false},
		{"option value", []string{"--config", "--json", "config", "show"}, false},
		{"disabled output", []string{"--json", "--json=false", "invalid"}, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			originalArgs, originalStderr := os.Args, os.Stderr
			t.Cleanup(func() {
				os.Args, os.Stderr = originalArgs, originalStderr
			})

			directory := t.TempDir()
			t.Chdir(directory)
			path := filepath.Join(directory, "stderr")
			output, err := os.Create(path)
			require.NoError(t, err)
			t.Cleanup(func() { _ = output.Close() })

			os.Args = append([]string{"nexa"}, scenario.args...)
			os.Stderr = output
			require.Equal(t, 1, run(context.Background()))

			var content []byte

			content, err = os.ReadFile(path)
			require.NoError(t, err)
			require.NotEmpty(t, content)
			require.Equal(t, scenario.json, json.Valid(content))
		})
	}
}
