package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	ghc "flavor-vault/internal/github"
	"flavor-vault/internal/vault"
)

// newInitCmd 初始化本地开发目录：.flavor-vault + 活动配置。
// 作者信息可选（提交时默认基于每个使用者的 GITHUB_TOKEN 自动识别；此处仅作显式覆盖）。
func newInitCmd() *cobra.Command {
	var (
		force       bool
		endpoint    string
		github      bool
		repo        string
		branch      string
		authorName  string
		authorEmail string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "对话式初始化配置（默认即可用，只需只读；编辑菜谱时才配置 GitHub）",
		Args:  cobra.NoArgs,
		Example: `  fv init                                                        # 对话式，回车即用默认（只读即可用）
  fv init --github --repo owner/recipes --branch recipes             # 非交互：一并配置编辑仓库
  fv init -c /path/to/config.yaml                                    # 在指定位置生成配置
  # 之后（AI 助手请显式 -c 指定配置）：
  #   读取  → fv list -c ~/.flavor-vault/config.yaml
  #   编辑  → fv add -c ~/.flavor-vault/config.yaml --repo owner/repo --branch recipes + GITHUB_TOKEN
  #           作者默认取自你的 GITHUB_TOKEN 对应 GitHub 账户`,
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
				// 默认写入用户主目录（~/.flavor-vault/config.yaml）：
				// AI 助手（OpenClaw 等）安装/调用方式各异，主目录是最稳定的位置
				dir = vault.HomeRoot()
				cfgPath = vault.HomeConfigPath()
			}

			// 对话式初始化：默认配置（只读）即可用；只有需要编辑菜谱时才询问更多信息。
			// 非交互（CI/AI）用 --github/--repo/--branch/--author-* 显式传参，不发起咨询。
			setupGitHub := github
			repoVal := strings.TrimSpace(repo)
			branchVal := strings.TrimSpace(branch)
			if isInteractive() {
				reader := newLineReader()
				fmt.Fprintln(cmd.OutOrStdout(), "Flavor Vault 🍲 将生成默认配置，读取菜谱即可用（无需任何额外信息）。")
				if !cmd.Flags().Changed("github") {
					if v, err := promptBool(reader, "需要新增/编辑/删除菜谱吗（需 GitHub 数据仓库；只读可跳过）", false); err == nil {
						setupGitHub = v
					}
				}
				if setupGitHub {
					if !cmd.Flags().Changed("repo") {
						def := ""
						if owner, r, err := ghc.ResolveRepo(""); err == nil {
							def = owner + "/" + r
						}
						if v, err := prompt(reader, "GitHub 数据仓库（owner/repo，菜谱存放处）", def); err == nil {
							repoVal = strings.TrimSpace(v)
						}
					}
					if !cmd.Flags().Changed("branch") {
						if v, err := prompt(reader, "数据分支", "recipes"); err == nil {
							branchVal = strings.TrimSpace(v)
						}
					}
					if !cmd.Flags().Changed("author-name") {
						if v, err := prompt(reader, "提交作者姓名（回车则提交时自动取 GitHub 账户）", gitConfigDefault("user.name")); err == nil {
							authorName = strings.TrimSpace(v)
						}
					}
					if !cmd.Flags().Changed("author-email") {
						if v, err := prompt(reader, "提交作者邮箱（回车则自动取 GitHub 账户）", gitConfigDefault("user.email")); err == nil {
							authorEmail = strings.TrimSpace(v)
						}
					}
				}
			}

			return initVault(cmd, dir, cfgPath, force, endpoint, authorName, authorEmail, setupGitHub, repoVal, branchVal)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "覆盖已存在的配置文件")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "默认数据 endpoint（可选）")
	cmd.Flags().BoolVar(&github, "github", false, "同时配置 GitHub 数据仓库（编辑菜谱用）")
	cmd.Flags().StringVar(&repo, "repo", "", "GitHub 数据仓库 owner/repo")
	cmd.Flags().StringVar(&branch, "branch", "", "数据分支（默认 recipes）")
	cmd.Flags().StringVar(&authorName, "author-name", "", "作者姓名（可选覆盖，默认自动取 GitHub 账户）")
	cmd.Flags().StringVar(&authorEmail, "author-email", "", "作者邮箱（可选覆盖）")
	return cmd
}

// initVault 创建 .flavor-vault 目录与活动配置
func initVault(cmd *cobra.Command, dir, cfgPath string, force bool, endpoint, authorName, authorEmail string, setupGitHub bool, repo, branch string) error {
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
		if err := os.WriteFile(cfgPath, []byte(configYAML(endpoint, authorName, authorEmail, repo, branch)), 0o644); err != nil {
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
	if setupGitHub {
		b := strings.TrimSpace(branch)
		if b == "" {
			b = "recipes"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  编辑：fv add/edit/rm（仓库 %s，分支 %s；需 GITHUB_TOKEN）\n", repo, b)
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  编辑（以后需要时）：运行 fv init --github，或在 fv add 时按提示补全 GitHub 信息")
	}
	fmt.Fprintln(cmd.OutOrStdout(), "  构建/发布：由 GitHub Actions 自动处理")
	return nil
}

// configYAML 生成活动配置（endpoint + github 块；作者仅在提供时写入）
func configYAML(endpoint, authorName, authorEmail, repo, branch string) string {
	var b strings.Builder
	b.WriteString("# Flavor Vault CLI 配置\n")
	if authorName != "" || authorEmail != "" {
		b.WriteString("author:\n")
		b.WriteString(fmt.Sprintf("    name: %q\n", authorName))
		b.WriteString(fmt.Sprintf("    email: %q\n", authorEmail))
	}
	b.WriteString(fmt.Sprintf("endpoint: %q\n", strings.TrimSpace(endpoint)))
	if strings.TrimSpace(repo) != "" {
		b.WriteString("github:\n")
		b.WriteString("    token: \"\"         # GitHub 令牌（建议用环境变量 GITHUB_TOKEN）\n")
		b.WriteString(fmt.Sprintf("    repo: %q\n", strings.TrimSpace(repo)))
		br := strings.TrimSpace(branch)
		if br == "" {
			br = "recipes"
		}
		b.WriteString(fmt.Sprintf("    branch: %q\n", br))
	} else {
		b.WriteString(`# github:
#     token: ""         # GitHub 令牌（建议用环境变量 GITHUB_TOKEN）
#     repo: ""          # 菜谱数据仓库 owner/repo；留空 = 当前仓库
#     branch: recipes   # 数据分支（默认 recipes）
`)
	}
	b.WriteString(`# author:
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
