package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/vault"
)

// newInitCmd 初始化本地开发目录（可选）：.flavor-vault/{cache} + 可选 config + .gitignore。
// 配置文件不是必需：读取用 --endpoint，编辑用 --repo/--branch + GITHUB_TOKEN。
func newInitCmd() *cobra.Command {
	var (
		force    bool
		endpoint string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化本地开发目录（可选）：.flavor-vault + CLI 配置示例",
		Args:  cobra.NoArgs,
		Example: `  fv init                                              # 创建 .flavor-vault 与 CLI 配置示例
  fv init -c /path/to/config.yaml                       # 在指定位置初始化
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
			return initVault(cmd, dir, cfgPath, force, endpoint)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "覆盖已存在的配置文件")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "默认数据 endpoint（写入配置示例，可选）")
	return cmd
}

// initVault 创建 .flavor-vault 目录与 CLI 配置示例
func initVault(cmd *cobra.Command, dir, cfgPath string, force bool, endpoint string) error {
	vaultDir := vault.VaultRoot(dir)
	for _, d := range []string{vaultDir, vault.CacheRoot(dir)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", d, err)
		}
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) || force {
		if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(cfgPath, []byte(sampleConfigYAML(endpoint)), 0o644); err != nil {
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

// sampleConfigYAML 生成 CLI 配置示例（只读 endpoint / 读写 github）
func sampleConfigYAML(endpoint string) string {
	return fmt.Sprintf(`# Flavor Vault CLI 配置示例（可选）
#
# 只读（查询菜谱）：只需 endpoint
# 读写（编辑菜谱）：配置 github（菜谱仓库 + 权限）；或用 --repo/--branch + GITHUB_TOKEN

endpoint: %q
# github:
#     token: ""         # GitHub 令牌（建议用环境变量 GITHUB_TOKEN）
#     repo: ""          # 菜谱数据仓库 owner/repo；留空 = 当前仓库
#     branch: recipes   # 数据分支（默认 recipes）
`, strings.TrimSpace(endpoint))
}

// isInteractive 判断标准输入是否为终端（决定是否发起交互式咨询）
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
