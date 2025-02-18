// Copyright (C) nexa. 2025-present.
//
// Created at 2025-02-17, by liasica

package gitlab

import (
	gitlab "gitlab.com/gitlab-org/api/client-go"
)

// LatestCommitID 获取分支最新的提交 SHA1 ID
// https://gitlab.cn/docs/jh/api/branches.html
func (g *Gitlab) LatestCommitID(repository, branch string) (id string, err error) {
	var b *gitlab.Branch
	b, _, err = g.client.Branches.GetBranch(repository, branch)
	if err != nil {
		return
	}
	id = b.Commit.ID
	return
}
