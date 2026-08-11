package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
	"flavor-vault/internal/vault"
)

// newInitCmd 初始化本地开发目录（可选）：.flavor-vault/{cache} + 可选 config + .gitignore。
// 配置文件不是必需：读取用 --endpoint，编辑用 --repo/--branch + GITHUB_TOKEN。
func newInitCmd() *cobra.Command {
	var (
		force    bool
		endpoint string
		assetDir string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化本地开发目录（可选；读取用 --endpoint，编辑用 --repo/--branch）",
		Args:  cobra.NoArgs,
		Example: `  fv init                                              # 创建 .flavor-vault 与默认配置
  fv init -c /path/to/config.yaml                       # 在指定位置初始化配置
  fv init --endpoint https://owner.github.io/repo/data  # 记录默认读取地址
  # 之后无需配置文件：
  #   读取  → fv list --endpoint <url>  /  fv search 词 --endpoint <url>
  #   编辑  → fv add --repo owner/repo --branch recipes（需 GITHUB_TOKEN）`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFlag, _ := cmd.Flags().GetString("config")
			var dir, cfgPath string
			if strings.TrimSpace(configFlag) != "" {
				root, cp, err := vault.ResolveWithConfig(configFlag)
				if err != nil {
					return err
				}
				dir = root
				cfgPath = cp
			} else {
				d, err := os.Getwd()
				if err != nil {
					return err
				}
				dir = d
				cfgPath = vault.ConfigPath(d)
			}

			cfg := models.DefaultConfig()
			if cmd.Flags().Changed("endpoint") {
				cfg.Endpoint = endpoint
			}
			if cmd.Flags().Changed("asset-dir") {
				cfg.AssetDir = assetDir
			}
			return initVault(cmd, dir, cfgPath, force, cfg)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "覆盖已存在的配置文件")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "默认数据 endpoint（记录到配置，可选）")
	cmd.Flags().StringVar(&assetDir, "asset-dir", ".flavor-vault/assets", "图片资源目录（相对项目根，可选）")
	return cmd
}

// initVault 创建 .flavor-vault 目录与可选配置
func initVault(cmd *cobra.Command, dir, cfgPath string, force bool, cfg *models.Config) error {
	vaultDir := vault.VaultRoot(dir)
	for _, d := range []string{vaultDir, vault.CacheRoot(dir)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", d, err)
		}
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) || force {
		if err := vault.SaveConfigAt(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✔ 已生成配置 %s\n", cfgPath)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "配置已存在，跳过: %s\n", cfgPath)
	}

	gitignore := filepath.Join(dir, ".gitignore")
	entries := []string{".flavor-vault/cache/", ".flavor-vault/push.lock", "/dist/", "web/node_modules/"}
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		content := ""
		for _, e := range entries {
			content += e + "\n"
		}
		_ = os.WriteFile(gitignore, []byte(content), 0o644)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔ Flavor Vault 初始化完成：%s\n", vaultDir)
	fmt.Fprintln(cmd.OutOrStdout(), "  读取：fv list/search/show --endpoint <url>")
	fmt.Fprintln(cmd.OutOrStdout(), "  编辑：fv add/edit/rm --repo <owner/repo> --branch <branch>（需 GITHUB_TOKEN）")
	fmt.Fprintln(cmd.OutOrStdout(), "  构建/发布：由 GitHub Actions 自动处理")
	return nil
}

// isInteractive 判断标准输入是否为终端（决定是否发起交互式咨询）
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
