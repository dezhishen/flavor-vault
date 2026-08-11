package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/vault"
)

// newInitCmd 初始化本地开发目录：.flavor-vault + 活动配置。
// 作者信息可选（提交时默认基于每个使用者的 GITHUB_TOKEN 自动识别；此处仅作显式覆盖）。
func newInitCmd() *cobra.Command {
	var (
		force       bool
		endpoint    string
		authorName  string
		authorEmail string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化本地开发目录：.flavor-vault + 配置（作者可选覆盖）",
		Args:  cobra.NoArgs,
		Example: `  fv init                                                        # 交互式（可选采集作者/endpoint）
  fv init --author-name "张三" --author-email "zhang@example.com"   # 显式作者覆盖（可选）
  fv init -c /path/to/config.yaml                                  # 在指定位置生成配置
  # 之后：
  #   读取  → fv list / fv search（默认端点 https://fv.sdniu.top/data）
  #   编辑  → fv add --repo owner/repo --branch recipes + GITHUB_TOKEN
  #           提交作者默认取自你的 GITHUB_TOKEN 对应 GitHub 账户`,
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
				cfgPath = vault.ConfigPath(d) // 活动配置：.flavor-vault/config.yaml
			}

			// 作者可选（交互式：默认取 git config，回车跳过；非交互：无 flag 则跳过）
			if isInteractive() {
				reader := newLineReader()
				if !cmd.Flags().Changed("author-name") {
					if v, err := prompt(reader, "作者姓名（回车跳过，提交时自动取 GitHub 账户）", gitConfigDefault("user.name")); err == nil {
						authorName = strings.TrimSpace(v)
					}
				}
				if !cmd.Flags().Changed("author-email") {
					if v, err := prompt(reader, "作者邮箱（回车跳过）", gitConfigDefault("user.email")); err == nil {
						authorEmail = strings.TrimSpace(v)
					}
				}
			}

			return initVault(cmd, dir, cfgPath, force, endpoint, authorName, authorEmail)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "覆盖已存在的配置文件")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "默认数据 endpoint（可选）")
	cmd.Flags().StringVar(&authorName, "author-name", "", "作者姓名（可选覆盖，默认自动取 GitHub 账户）")
	cmd.Flags().StringVar(&authorEmail, "author-email", "", "作者邮箱（可选覆盖）")
	return cmd
}

// initVault 创建 .flavor-vault 目录与活动配置
func initVault(cmd *cobra.Command, dir, cfgPath string, force bool, endpoint, authorName, authorEmail string) error {
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
		if err := os.WriteFile(cfgPath, []byte(configYAML(endpoint, authorName, authorEmail)), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✔ 已生成配置 %s\n", cfgPath)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "配置已存在，跳过: %s（如需更新，运行 fv config set ...）\n", cfgPath)
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
	if authorName != "" || authorEmail != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  作者覆盖：%s <%s>\n", authorName, authorEmail)
	}
	fmt.Fprintln(cmd.OutOrStdout(), "  读取：fv list/search/show（默认端点 https://fv.sdniu.top/data）")
	fmt.Fprintln(cmd.OutOrStdout(), "  编辑：fv add/edit/rm --repo <owner/repo> --branch <branch>（需 GITHUB_TOKEN）")
	fmt.Fprintln(cmd.OutOrStdout(), "  构建/发布：由 GitHub Actions 自动处理")
	return nil
}

// configYAML 生成活动配置（endpoint + github 占位；作者仅在提供时写入）
func configYAML(endpoint, authorName, authorEmail string) string {
	var b strings.Builder
	b.WriteString("# Flavor Vault CLI 配置\n")
	if authorName != "" || authorEmail != "" {
		b.WriteString("author:\n")
		b.WriteString(fmt.Sprintf("    name: %q\n", authorName))
		b.WriteString(fmt.Sprintf("    email: %q\n", authorEmail))
	}
	b.WriteString(fmt.Sprintf("endpoint: %q\n", strings.TrimSpace(endpoint)))
	b.WriteString(`# github:
#     token: ""         # GitHub 令牌（建议用环境变量 GITHUB_TOKEN）
#     repo: ""          # 菜谱数据仓库 owner/repo；留空 = 当前仓库
#     branch: recipes   # 数据分支（默认 recipes）
# author:
#     name: ""          # 可选：提交作者覆盖；不配则自动取 GITHUB_TOKEN 对应 GitHub 账户
#     email: ""
`)
	return b.String()
}

// gitConfigDefault 读取 git config 某键值（供交互默认值；无则空）
func gitConfigDefault(key string) string {
	out, err := exec.Command("git", "config", "--get", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// isInteractive 判断标准输入是否为终端（决定是否发起交互式咨询）
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
