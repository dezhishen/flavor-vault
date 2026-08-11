package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
	"flavor-vault/internal/vault"
)

// resolveProject 解析项目上下文（项目根 + 配置文件路径）。
// 配置可选：--config 优先；否则若有 .flavor-vault 则使用；都没有则用当前目录（返回默认配置）。
func resolveProject(cmd *cobra.Command) (projectRoot, configPath string, found bool) {
	configFlag, _ := cmd.Flags().GetString("config")
	if strings.TrimSpace(configFlag) != "" {
		root, cp, err := vault.ResolveWithConfig(configFlag)
		if err != nil {
			return ".", configFlag, false
		}
		return root, cp, true
	}
	if root, err := vault.FindRoot(); err == nil {
		return root, vault.ConfigPath(root), true
	}
	d, _ := os.Getwd()
	return d, "", false
}

// loadProjectConfig 解析配置。配置文件可选：
//   - 有配置文件（--config 或 .flavor-vault/config.yaml）→ 加载；
//   - 没有 → 用默认配置；
//   - 全局标志/环境变量覆盖：--endpoint/FV_ENDPOINT、--repo/FV_REPO、--branch/FV_BRANCH。
//
// 返回 (配置, 项目根, 配置文件路径, 错误)。
func loadProjectConfig(cmd *cobra.Command) (*models.Config, string, string, error) {
	projectRoot, configPath, found := resolveProject(cmd)
	var cfg *models.Config
	if found {
		var err error
		cfg, err = vault.LoadConfigAt(configPath)
		if err != nil {
			return nil, "", "", err
		}
	} else {
		cfg = models.DefaultConfig()
	}

	// 标志/环境覆盖（编辑参数：--repo/--branch；读取参数：--endpoint）
	if v, _ := cmd.Flags().GetString("endpoint"); strings.TrimSpace(v) != "" {
		cfg.Endpoint = strings.TrimSpace(v)
	} else if e := os.Getenv("FV_ENDPOINT"); strings.TrimSpace(e) != "" {
		cfg.Endpoint = strings.TrimSpace(e)
	}
	if v, _ := cmd.Flags().GetString("repo"); strings.TrimSpace(v) != "" {
		cfg.GitHub.Repo = strings.TrimSpace(v)
	} else if e := os.Getenv("FV_REPO"); strings.TrimSpace(e) != "" {
		cfg.GitHub.Repo = strings.TrimSpace(e)
	}
	if v, _ := cmd.Flags().GetString("branch"); strings.TrimSpace(v) != "" {
		cfg.GitHub.Branch = strings.TrimSpace(v)
	} else if e := os.Getenv("FV_BRANCH"); strings.TrimSpace(e) != "" {
		cfg.GitHub.Branch = strings.TrimSpace(e)
	}

	return cfg, projectRoot, configPath, nil
}

// recipesDir 返回构建用到的本地菜谱目录（CI checkout / 本地检出）
func recipesDir(cfg *models.Config, projectRoot string) string {
	return vault.ResolveRecipesDir(projectRoot, cfg)
}

// assetDirFor 返回本地图片资源目录（编辑时暂存待提交的图片）
func assetDirFor(cfg *models.Config, projectRoot string) string {
	return vault.ResolveAssetDir(projectRoot, cfg)
}
