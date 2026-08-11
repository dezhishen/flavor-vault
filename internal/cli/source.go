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

// newSourceCmd 管理外部菜谱数据源（引用本项目外部的菜谱仓库）
func newSourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "管理外部菜谱数据源（引用其他仓库的菜谱，维护者聚合/使用者浏览）",
	}
	cmd.AddCommand(
		newSourceAddCmd(),
		newSourceListCmd(),
		newSourceRemoveCmd(),
		newSourcePullCmd(),
	)
	return cmd
}

// allRecipeDirs 返回构建/查找用到的全部菜谱目录：本地 + 已克隆的外部数据源
func allRecipeDirs(cfg *models.Config, projectRoot string) []string {
	dirs := []string{recipesDir(cfg, projectRoot)}
	for _, s := range cfg.Sources {
		name := strings.TrimSpace(s.Name)
		if name == "" {
			continue
		}
		d := vault.SourceRecipesDir(projectRoot, name)
		if _, err := os.Stat(d); err == nil {
			dirs = append(dirs, d)
		}
	}
	return dirs
}

func newSourceAddCmd() *cobra.Command {
	var branch string
	cmd := &cobra.Command{
		Use:   "add <name> <repo-url>",
		Short: "添加外部菜谱数据源（git 仓库 URL）",
		Args:  cobra.ExactArgs(2),
		Example: `  fv source add community git@github.com:someone/recipes.git
  fv source add community https://github.com/someone/recipes.git --branch main`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, cfgPath, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("数据源名称不能为空")
			}
			for _, s := range cfg.Sources {
				if s.Name == name {
					return fmt.Errorf("数据源 %q 已存在（可用 fv source remove 移除后重加）", name)
				}
			}
			if branch == "" {
				branch = "recipes"
			}
			cfg.Sources = append(cfg.Sources, models.SourceConfig{Name: name, Repo: args[1], Branch: branch})
			if err := vault.SaveConfigAt(cfgPath, cfg); err != nil {
				return err
			}
			_ = projectRoot
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已添加数据源 %s -> %s（分支 %s）\n", name, args[1], branch)
			fmt.Fprintln(cmd.OutOrStdout(), "  运行 fv source pull 拉取菜谱，fv build 合并构建。")
			return nil
		},
	}
	cmd.Flags().StringVar(&branch, "branch", "recipes", "外部仓库的菜谱分支（默认 recipes）")
	return cmd
}

func newSourceListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出已配置的外部菜谱数据源",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			if len(cfg.Sources) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(未配置外部数据源，运行 fv source add <name> <url> 添加)")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "外部菜谱数据源:")
			for _, s := range cfg.Sources {
				dir := vault.SourceCloneDir(projectRoot, s.Name)
				status := "未拉取"
				if _, err := os.Stat(dir); err == nil {
					status = "已拉取"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-16s %-8s 分支 %-10s %s\n", s.Name, status, s.Branch, s.Repo)
			}
			return nil
		},
	}
}

func newSourceRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "移除外部菜谱数据源（可保留已克隆的数据）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, cfgPath, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			name := args[0]
			found := false
			out := cfg.Sources[:0]
			for _, s := range cfg.Sources {
				if s.Name == name {
					found = true
					continue
				}
				out = append(out, s)
			}
			cfg.Sources = out
			if !found {
				return fmt.Errorf("数据源 %q 不存在", name)
			}
			if err := vault.SaveConfigAt(cfgPath, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已移除数据源 %s（本地克隆可手动删除 %s）\n", name, vault.SourceCloneDir(projectRootForRemove(cmd), name))
			return nil
		},
	}
}

func newSourcePullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull [name...]",
		Short: "克隆/更新外部菜谱数据源到本地",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			if len(cfg.Sources) == 0 {
				return fmt.Errorf("未配置外部数据源，先运行 fv source add <name> <url>")
			}
			want := map[string]bool{}
			for _, a := range args {
				want[a] = true
			}
			for _, s := range cfg.Sources {
				if len(want) > 0 && !want[s.Name] {
					continue
				}
				if err := pullSource(projectRoot, s); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "⚠ 数据源 %s 拉取失败: %v\n", s.Name, err)
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✔ 数据源 %s 已更新\n", s.Name)
			}
			return nil
		},
	}
}

// pullSource 克隆或更新单个外部数据源
func pullSource(projectRoot string, s models.SourceConfig) error {
	dir := vault.SourceCloneDir(projectRoot, s.Name)
	branch := s.Branch
	if branch == "" {
		branch = "recipes"
	}
	env := append(os.Environ(), "GIT_SSH_COMMAND=ssh -o BatchMode=yes -o StrictHostKeyChecking=accept-new")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(vault.SourcesDir(projectRoot), 0o755); err != nil {
			return err
		}
		// 先试配置分支，失败则回退默认分支
		if out, err := gitRunIn(projectRoot, env, "clone", "--branch", branch, "--depth", "1", s.Repo, dir); err != nil {
			_ = os.RemoveAll(dir)
			if out2, err2 := gitRunIn(projectRoot, env, "clone", "--depth", "1", s.Repo, dir); err2 != nil {
				return fmt.Errorf("git clone 失败: %w\n%s\n%s", err2, out, out2)
			}
		}
		return nil
	}
	// 已有克隆 → 拉取更新
	if out, err := gitRunIn(dir, env, "pull", "--ff-only"); err != nil {
		return fmt.Errorf("git pull 失败: %w\n%s", err, out)
	}
	return nil
}

func gitRunIn(dir string, env []string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = env
	out, err := c.CombinedOutput()
	return string(out), err
}

// projectRootForRemove 仅用于 remove 命令打印克隆目录（避免多余解析）
func projectRootForRemove(cmd *cobra.Command) string {
	root, _, err := resolveProject(cmd)
	if err != nil {
		return filepath.Join(".", ".flavor-vault")
	}
	return root
}
