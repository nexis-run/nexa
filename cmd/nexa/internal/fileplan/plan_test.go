package fileplan

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

type checkpointContext struct {
	context.Context
	checks  int
	onCheck func(int)
}

func (ctx *checkpointContext) Err() error {
	ctx.checks++
	ctx.onCheck(ctx.checks)

	return ctx.Context.Err()
}

func writeFixture(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, []byte(content), mode))
}

func requireContent(t *testing.T, path, expected string) {
	t.Helper()

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, expected, string(content))
}

func requireNoStagedFiles(t *testing.T, root string) {
	t.Helper()

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		require.NoError(t, err)
		require.NotContains(t, entry.Name(), ".nexa-", path)

		return nil
	})
	require.NoError(t, err)
}

func TestPlanPreflightsWholeBatch(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing.go")
	newFile := filepath.Join(root, "new", "nested", "created.go")
	writeFixture(t, existing, "original", 0600)
	_, err := New(
		File{Path: newFile, Content: []byte("created")},
		File{Path: existing, Content: []byte("replacement")},
	)
	require.Error(t, err)
	require.NoDirExists(t, filepath.Join(root, "new"))
	requireContent(t, existing, "original")
	requireNoStagedFiles(t, root)
}

func TestPlanProtectsConcurrentEdits(t *testing.T) {
	for _, scenario := range []struct {
		name             string
		checkpoint       int
		permissionChange bool
	}{
		{name: "before apply", checkpoint: 0},
		{name: "during apply", checkpoint: 5},
		{name: "permissions before rollback", checkpoint: 5, permissionChange: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			root := t.TempDir()
			first := filepath.Join(root, "first.go")
			second := filepath.Join(root, "second.go")
			writeFixture(t, first, "first-original", 0640)
			writeFixture(t, second, "second-original", 0600)
			plan, err := New(
				File{Path: first, Content: []byte("first-new"), Overwrite: true},
				File{Path: second, Content: []byte("second-new"), Overwrite: true},
			)
			require.NoError(t, err)

			if scenario.checkpoint == 0 {
				writeFixture(t, second, "external-edit", 0600)
			}

			ctx := &checkpointContext{Context: context.Background(), onCheck: func(current int) {
				if current == scenario.checkpoint {
					if scenario.permissionChange {
						require.NoError(t, os.Chmod(first, 0600))
					}

					writeFixture(t, second, "external-edit", 0600)
				}
			}}
			err = plan.Apply(ctx)
			require.ErrorContains(t, err, "发生变化")

			if scenario.permissionChange {
				require.ErrorContains(t, err, "未回滚")
				requireContent(t, first, "first-new")
				var info os.FileInfo

				info, err = os.Stat(first)
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0600), info.Mode().Perm())
			} else {
				requireContent(t, first, "first-original")
			}

			requireContent(t, second, "external-edit")
			requireNoStagedFiles(t, root)
		})
	}
}

func TestPlanCancellationCleansUp(t *testing.T) {
	for _, checkpoint := range []int{1, 3, 5} {
		t.Run(strconv.Itoa(checkpoint), func(t *testing.T) {
			root := t.TempDir()
			first := filepath.Join(root, "new", "nested", "first.go")
			second := filepath.Join(root, "new", "other", "second.go")
			plan, err := New(
				File{Path: first, Content: []byte("first")},
				File{Path: second, Content: []byte("second")},
			)
			require.NoError(t, err)

			parent, cancel := context.WithCancel(context.Background())
			defer cancel()

			ctx := &checkpointContext{Context: parent, onCheck: func(current int) {
				if current == checkpoint {
					cancel()
				}
			}}
			err = plan.Apply(ctx)
			require.ErrorIs(t, err, context.Canceled)
			require.NoDirExists(t, filepath.Join(root, "new"))
			requireNoStagedFiles(t, root)
		})
	}
}

func TestPlanRollsBackMixedChangesOnWriteFailure(t *testing.T) {
	root := t.TempDir()
	updated := filepath.Join(root, "updated.go")
	deleted := filepath.Join(root, "deleted.go")
	created := filepath.Join(root, "nested", "created.go")
	failed := filepath.Join(root, "failed.go")
	writeFixture(t, updated, "update-original", 0640)
	writeFixture(t, deleted, "delete-original", 0600)
	writeFixture(t, failed, "failure-original", 0644)
	plan, err := New(
		File{Path: updated, Content: []byte("updated"), Overwrite: true},
		File{Path: deleted, Remove: true, Overwrite: true},
		File{Path: created, Content: []byte("created")},
		File{Path: failed, Content: []byte("failure-new"), Overwrite: true},
	)
	require.NoError(t, err)

	ctx := &checkpointContext{Context: context.Background(), onCheck: func(current int) {
		if current == 9 {
			require.NoError(t, os.Remove(plan.files[3].staged))
		}
	}}
	err = plan.Apply(ctx)
	require.ErrorIs(t, err, os.ErrNotExist)
	requireContent(t, updated, "update-original")
	requireContent(t, deleted, "delete-original")
	requireContent(t, failed, "failure-original")
	require.NoDirExists(t, filepath.Join(root, "nested"))
	requireNoStagedFiles(t, root)

	var info os.FileInfo

	info, err = os.Stat(updated)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0640), info.Mode().Perm())
}
