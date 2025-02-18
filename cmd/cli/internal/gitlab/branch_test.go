// Copyright (C) nexa. 2025-present.
//
// Created at 2025-02-17, by liasica

package gitlab

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLatestCommit(t *testing.T) {
	g, err := New(os.Getenv("GITLAB_ACCESS_TOKEN"))
	require.NoError(t, err)

	var id string
	id, err = g.LatestCommitID(DefaultRepository, DefaultBranch)
	t.Logf("latest commit id: %s", id)
}
