package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
	"flavor-vault/internal/vault"
)

// newSourceCmd 管理唯一菜谱数据源（GitHub，与代码隔离）。
// CLI 的增删改/推送只作用于该数据源，绝不写入程序代码所在分支。
func newSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "管理唯一菜谱数据源（GitHub；维护者配置，CLI 只操作该数据源，不触碰代码分支）",
	}
	cmd.AddCommand(
		newSourceShowCmd(),
		newSourceSetCmd(),
		newSourcePullCmd(),
		newSourceRemoveCmd(),
	)
	return cmd
}

// allRecipeDirs 返回构建/查找用到的全部菜谱目录：唯一数据源的检出目录
func allRecipeDirs(cfg *models.Config, projectRoot string) []string {
	return []string{recipesDir(cfg, projectRoot)}
}

func newSourceShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "显示菜谱数据源与 endpoint 配置",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			repo := strings.TrimSpace(cfg.Source.Repo)
			branch := strings.TrimSpace(cfg.Source.Branch)
			if cfg.Maintainer() {
				if repo == "" {
					repo = "（当前仓库）"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "菜谱数据源: %s（分支 %s）\n", repo, orDash(branch))
				status := "未检出"
				if _, err := os.Stat(vault.RecipesWorktree(projectRoot)); err == nil {
					status = "已检出（worktree）"
				} else if _, err := os.Stat(vault.SourceDir(projectRoot)); err == nil {
					status = "已检出（.flavor-vault/source）"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "本地检出: %s\n", status)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "菜谱数据源: 未配置（使用者模式，通过 endpoint 读取）")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "endpoint: %s\n", orDash(cfg.Endpoint))
			fmt.Fprintln(cmd.OutOrStdout(), "  （未配置时使用默认：本地 dist/data；构建时可用 FV_ENDPOINT 注入实际部署地址）")
			return nil
		},
	}
}

func newSourceSetCmd() *cobra.Command {
	var (
		sameRepo bool
		branch   string
	)
	cmd := &cobra.Command{
		Use:   "set [repo]",
		Short: "配置唯一菜谱数据源（GitHub 仓库；--same-repo 表示当前仓库的独立数据分支）",
		Example: `  fv source set --same-repo --branch recipes   # 当前仓库的 recipes 分支作为数据源
  fv source set owner/recipes --branch recipes # 独立仓库作为数据源
  fv source pull                                # 检出/更新数据源到本地`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, cfgPath, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			repo := ""
			if len(args) == 1 {
				repo = strings.TrimSpace(args[0])
			}
			if sameRepo {
				repo = ""
			}
			if repo == "" && strings.TrimSpace(cfg.Source.Repo) == "" && branch == "" && !sameRepo {
				return fmt.Errorf("请提供数据源仓库（如 owner/recipes）或 --same-repo")
			}
			if branch == "" {
				branch = "recipes"
			}
			cfg.Source = models.SourceConfig{Repo: repo, Branch: branch}
			if err := vault.SaveConfigAt(cfgPath, cfg); err != nil {
				return err
			}
			if repo == "" {
				fmt.Fprintf(cmd.OutOrStdout(), "✔ 已配置数据源：当前仓库 %s 分支\n", branch)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "✔ 已配置数据源：%s（分支 %s）\n", repo, branch)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "  运行 fv source pull 检出到本地，之后 fv add / fv build / fv gh push 均只作用于该数据源。")
			return nil
		},
	}
	cmd.Flags().BoolVar(&sameRepo, "same-repo", false, "使用当前仓库的独立数据分支（repo 留空）")
	cmd.Flags().StringVar(&branch, "branch", "", "数据源分支（默认 recipes）")
	return cmd
}

func newSourcePullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "检出/更新菜谱数据源到本地（.flavor-vault/source 或 .recipes worktree）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			if !cfg.Maintainer() {
				return fmt.Errorf("未配置菜谱数据源，先运行 fv source set <repo>（或 --same-repo）")
			}
			if err := pullSource(cmd, projectRoot, cfg); err != nil {
				return err
			}
			return nil
		},
	}
}

func newSourceRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove",
		Short: "移除菜谱数据源配置（保留本地检出）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, cfgPath, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			if !cfg.Maintainer() {
				return fmt.Errorf("未配置菜谱数据源")
			}
			cfg.Source = models.SourceConfig{}
			if err := vault.SaveConfigAt(cfgPath, cfg); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "✔ 已移除数据源配置（本地检出可手动删除）")
			return nil
		},
	}
}

// pullSource 检出/更新唯一数据源。
// repo 为空（--same-repo）时用同仓库独立分支的 worktree；否则克隆到 .flavor-vault/source。
func pullSource(cmd *cobra.Command, projectRoot string, cfg *models.Config) error {
	repo := strings.TrimSpace(cfg.Source.Repo)
	branch := strings.TrimSpace(cfg.Source.Branch)
	if branch == "" {
		branch = "recipes"
	}
	env := append(os.Environ(), "GIT_SSH_COMMAND=ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new")

	// 同仓库独立分支：更新 worktree
	if repo == "" {
		wt := vault.RecipesWorktree(projectRoot)
		if _, err := os.Stat(filepath.Join(wt, ".git")); os.IsNotExist(err) {
			return fmt.Errorf("尚未初始化同仓库数据分支 worktree（%s），请用 fv init --maintain 或 fv source set <repo> 指定独立仓库", wt)
		}
		if out, err := gitRun(wt, env, "pull", "--ff-only"); err != nil {
			return fmt.Errorf("更新 worktree 失败: %w\n%s", err, out)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✔ 数据源已更新（worktree %s）\n", wt)
		return nil
	}

	// 独立仓库：克隆/更新到 .flavor-vault/source
	dir := vault.SourceDir(projectRoot)
	if _, err := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(err) {
		if err := os.MkdirAll(vault.VaultRoot(projectRoot), 0o755); err != nil {
			return err
		}
		if out, err := gitRun(projectRoot, env, "clone", "--branch", branch, "--depth", "1", repo, dir); err != nil {
			_ = os.RemoveAll(dir)
			return fmt.Errorf("git clone %s 失败: %w\n%s", repo, err, out)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✔ 已检出数据源 %s -> %s\n", repo, dir)
		return nil
	}
	if out, err := gitRun(dir, env, "checkout", branch); err != nil {
		return fmt.Errorf("切换分支 %s 失败: %w\n%s", branch, err, out)
	}
	if out, err := gitRun(dir, env, "pull", "--ff-only", "origin", branch); err != nil {
		return fmt.Errorf("更新数据源失败: %w\n%s", err, out)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✔ 数据源已更新 %s（%s）\n", repo, branch)
	return nil
}

// gitRunIn 在指定目录执行 git 命令
func gitRunIn(dir string, env []string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = env
	out, err := c.CombinedOutput()
	return string(out), err
}
