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
		assetDir        string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化 .flavor-vault/ 及默认 config（交互式咨询菜谱数据配置）",
		Args:  cobra.NoArgs,
		Example: `  fv init                                          # 交互式：咨询自定义数据/独立分支
  fv init --separate-recipes                        # 菜谱数据放到独立 recipes 分支
  fv init --asset-dir custom/assets                 # 自定义图片资源目录
  fv init -c /path/to/config.yaml                   # 在指定位置初始化配置`,
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

			cfg := models.DefaultConfig()

			// 交互式咨询（仅终端；CI/管道自动跳过，使用参数默认值）
			if isInteractive() {
				reader := newLineReader()

				// 1. 自定义菜谱数据：图片资源目录
				if !cmd.Flags().Changed("asset-dir") {
					if ok, _ := promptBool(reader, "自定义菜谱数据（设置图片资源目录）?", false); ok {
						if v, err := prompt(reader, "图片资源目录（相对项目根）", cfg.AssetDir); err == nil && strings.TrimSpace(v) != "" {
							cfg.AssetDir = strings.TrimSpace(v)
						}
					}
				}

				// 2. 菜谱数据是否放到独立分支（数据仓库，可 fork/私有化）
				if !cmd.Flags().Changed("separate-recipes") {
					ok, _ := promptBool(reader, "将菜谱数据放到独立分支（数据仓库，可 fork/私有化）?", false)
					separateRecipes = ok
					if ok && !cmd.Flags().Changed("recipes-branch") {
						if v, err := prompt(reader, "菜谱独立分支名", "recipes"); err == nil && strings.TrimSpace(v) != "" {
							recipesBranch = strings.TrimSpace(v)
						}
					}
				}
			}

			// 参数显式覆盖
			if cmd.Flags().Changed("asset-dir") {
				cfg.AssetDir = assetDir
			}
			if separateRecipes {
				cfg.GitHub.RecipesBranch = recipesBranch
			}

			return initVault(cmd, dir, cfgPath, force, cfg, separateRecipes, recipesBranch)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "覆盖已存在的配置文件")
	cmd.Flags().BoolVar(&separateRecipes, "separate-recipes", false, "菜谱数据放到独立分支（默认 recipes）")
	cmd.Flags().StringVar(&recipesBranch, "recipes-branch", "recipes", "菜谱独立分支名（配合 --separate-recipes）")
	cmd.Flags().StringVar(&assetDir, "asset-dir", ".flavor-vault/assets", "图片资源目录（相对项目根）")
	return cmd
}

// initVault 在指定项目根目录初始化 vault 结构，配置写入 cfgPath
func initVault(cmd *cobra.Command, dir, cfgPath string, force bool, cfg *models.Config, separateRecipes bool, recipesBranch string) error {
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

// isInteractive 判断标准输入是否为终端（决定是否发起交互式咨询）
func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
