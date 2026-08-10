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
