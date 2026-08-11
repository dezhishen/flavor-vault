package cli

import (
	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
	"flavor-vault/internal/vault"
)

// resolveProject 解析项目上下文（项目根 + 配置文件路径），
// 优先使用全局 --config/-c 参数，否则自动查找 .flavor-vault。
func resolveProject(cmd *cobra.Command) (projectRoot, configPath string, err error) {
	configFlag, _ := cmd.Flags().GetString("config")
	return vault.ResolveContext(configFlag)
}

// loadProjectConfig 解析项目上下文并加载配置。
// 返回 (配置, 项目根, 配置文件路径, 错误)。
func loadProjectConfig(cmd *cobra.Command) (*models.Config, string, string, error) {
	projectRoot, configPath, err := resolveProject(cmd)
	if err != nil {
		return nil, "", "", err
	}
	cfg, err := vault.LoadConfigAt(configPath)
	if err != nil {
		return nil, "", "", err
	}
	return cfg, projectRoot, configPath, nil
}

// recipesDir 返回实际使用的菜谱目录（考虑独立菜谱分支的本地 worktree）
func recipesDir(cfg *models.Config, projectRoot string) string {
	return vault.ResolveRecipesDir(projectRoot, cfg)
}

// assetDirFor 返回实际使用的图片资源目录（考虑独立菜谱分支的 worktree）
func assetDirFor(cfg *models.Config, projectRoot string) string {
	return vault.ResolveAssetDir(projectRoot, cfg)
}
