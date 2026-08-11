package cli

import (
	"flavor-vault/internal/models"
)

// allRecipeDirs 返回构建用到的本地菜谱目录（CI checkout / 本地检出）。
// 编辑（add/edit/rm）已改为经 GitHub API 操作数据源分支，不再需要本地克隆/worktree。
func allRecipeDirs(cfg *models.Config, projectRoot string) []string {
	return []string{recipesDir(cfg, projectRoot)}
}
