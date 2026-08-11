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

func newInitCmd() *cobra.Command {
	var (
		force           bool
		separateRecipes bool
		recipesBranch   string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化 .flavor-vault/ 及默认 config（支持 -c 配置位置、--separate-recipes 独立菜谱分支）",
		Args:  cobra.NoArgs,
		Example: `  fv init                                          # 在当前目录初始化
  fv init -c /path/to/config.yaml                   # 在指定位置初始化配置
  fv init --separate-recipes                        # 菜谱数据放到独立 recipes 分支
  fv init --separate-recipes --recipes-branch data  # 自定义菜谱分支名`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 目标目录与配置路径：优先 --config 指定的配置文件位置
			configFlag, _ := cmd.Flags().GetString("config")
			var dir, cfgPath string
			if strings.TrimSpace(configFlag) != "" {
				root, cp, err := vault.ResolveWithConfig(configFlag)
				if err != nil {
					return err
				}
				dir = root
				cfgPath = cp // 配置写入 -c 指定的确切路径
			} else {
				d, err := os.Getwd()
				if err != nil {
					return err
				}
				dir = d
				cfgPath = vault.ConfigPath(d)
			}
			return initVault(cmd, dir, cfgPath, force, separateRecipes, recipesBranch)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "覆盖已存在的配置文件")
	cmd.Flags().BoolVar(&separateRecipes, "separate-recipes", false, "菜谱数据放到独立分支（默认 recipes）")
	cmd.Flags().StringVar(&recipesBranch, "recipes-branch", "recipes", "菜谱独立分支名（配合 --separate-recipes）")
	return cmd
}

// initVault 在指定项目根目录初始化 vault 结构，配置写入 cfgPath
func initVault(cmd *cobra.Command, dir, cfgPath string, force, separateRecipes bool, recipesBranch string) error {
	vaultDir := vault.VaultRoot(dir)

	// 创建目录
	for _, d := range []string{
		vaultDir,
		vault.RecipesDir(dir),
		vault.CacheRoot(dir),
	} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", d, err)
		}
	}

	// 写入默认配置（若不存在或 --force）
	cfg := models.DefaultConfig()
	if separateRecipes {
		cfg.GitHub.RecipesBranch = recipesBranch
	}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) || force {
		if err := vault.SaveConfigAt(cfgPath, cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✔ 已生成配置 %s\n", cfgPath)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "配置已存在，跳过: %s\n", cfgPath)
	}

	// 添加 .gitignore（忽略缓存与构建产物）
	gitignore := filepath.Join(dir, ".gitignore")
	entries := []string{".flavor-vault/cache/", ".flavor-vault/push.lock", "/dist/", "web/node_modules/"}
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		content := ""
		for _, e := range entries {
			content += e + "\n"
		}
		if err := os.WriteFile(gitignore, []byte(content), 0o644); err != nil {
			return err
		}
	}

	// 生成示例菜谱（若 recipes 为空），作为独立分支的种子
	recipesDir := vault.RecipesDir(dir)
	if countJSON(recipesDir) == 0 {
		if err := writeSampleRecipe(recipesDir); err != nil {
			return err
		}
	}

	// 生成示例封面资源（图片/外链由菜谱 JSON 引用）
	if err := writeSampleAssets(vault.ResolveAssetDir(dir, cfg)); err != nil {
		return err
	}

	// 独立菜谱分支：创建"数据仓库"分支 + worktree + 忽略配置
	if separateRecipes {
		if err := setupRecipesBranch(cmd, dir, recipesBranch, cfg); err != nil {
			return err
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔ Flavor Vault 初始化完成：%s\n", vaultDir)
	fmt.Fprintf(cmd.OutOrStdout(), "  配置: %s\n", cfgPath)
	if !separateRecipes {
		fmt.Fprintln(cmd.OutOrStdout(), "运行 fv add 添加菜谱，fv build 构建站点。")
	}
	return nil
}

func countJSON(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			n++
		}
	}
	return n
}
