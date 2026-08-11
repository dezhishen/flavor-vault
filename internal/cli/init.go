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
		force        bool
		maintain     bool
		sourceRepo   string
		sourceBranch string
		endpoint     string
		assetDir     string
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化 .flavor-vault/ 及默认 config（交互式咨询维护者/使用者模式）",
		Args:  cobra.NoArgs,
		Example: `  fv init                                                  # 交互式咨询
  fv init --maintain --source-branch recipes                # 维护者（当前仓库独立数据分支）
  fv init --maintain --source-repo owner/recipes            # 维护者（独立数据仓库）
  fv init --endpoint https://owner.github.io/repo/data      # 使用者（只读查询）
  fv init -c /path/to/config.yaml                           # 在指定位置初始化配置`,
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

				// 1. 维护者 or 使用者
				if !cmd.Flags().Changed("maintain") {
					ok, _ := promptBool(reader, "是否维护菜谱（配置 GitHub 菜谱数据源）?", false)
					maintain = ok
				}

				if maintain {
					// 2. 数据源仓库（回车=当前仓库的独立数据分支）
					if !cmd.Flags().Changed("source-repo") {
						if v, err := prompt(reader, "菜谱数据源仓库（owner/repo，回车=当前仓库）", ""); err == nil && strings.TrimSpace(v) != "" {
							sourceRepo = strings.TrimSpace(v)
						}
					}
					if !cmd.Flags().Changed("source-branch") {
						if v, err := prompt(reader, "数据源分支（默认 recipes）", "recipes"); err == nil && strings.TrimSpace(v) != "" {
							sourceBranch = strings.TrimSpace(v)
						}
					}
					// 3. 图片资源目录
					if !cmd.Flags().Changed("asset-dir") {
						if v, err := prompt(reader, "图片资源目录（相对项目根）", cfg.AssetDir); err == nil && strings.TrimSpace(v) != "" {
							cfg.AssetDir = strings.TrimSpace(v)
						}
					}
				} else {
					// 4. 使用者：endpoint（回车用默认/本地）
					if !cmd.Flags().Changed("endpoint") {
						if v, err := prompt(reader, "数据 endpoint（回车用默认/本地）", ""); err == nil && strings.TrimSpace(v) != "" {
							endpoint = strings.TrimSpace(v)
						}
					}
				}
			}

			// 参数显式覆盖
			if maintain {
				if cmd.Flags().Changed("source-repo") {
					sourceRepo = strings.TrimSpace(sourceRepo)
				}
				if cmd.Flags().Changed("source-branch") {
					sourceBranch = strings.TrimSpace(sourceBranch)
				}
				if sourceBranch == "" {
					sourceBranch = "recipes"
				}
				cfg.Source = models.SourceConfig{Repo: sourceRepo, Branch: sourceBranch}
			}
			if cmd.Flags().Changed("asset-dir") {
				cfg.AssetDir = assetDir
			}
			if cmd.Flags().Changed("endpoint") {
				cfg.Endpoint = endpoint
			}

			return initVault(cmd, dir, cfgPath, force, cfg, maintain, sourceBranch)
		},
	}
	cmd.Flags().BoolVarP(&force, "force", "f", false, "覆盖已存在的配置文件")
	cmd.Flags().BoolVar(&maintain, "maintain", false, "维护者模式：配置 GitHub 菜谱数据源（CLI 只操作该数据源）")
	cmd.Flags().StringVar(&sourceRepo, "source-repo", "", "菜谱数据源仓库（owner/repo 或 URL；留空=当前仓库的独立数据分支）")
	cmd.Flags().StringVar(&sourceBranch, "source-branch", "recipes", "数据源分支（默认 recipes）")
	cmd.Flags().StringVar(&endpoint, "endpoint", "", "数据 endpoint（使用者模式；留空用默认/本地）")
	cmd.Flags().StringVar(&assetDir, "asset-dir", ".flavor-vault/assets", "图片资源目录（相对项目根）")
	return cmd
}

// initVault 在指定项目根目录初始化 vault 结构，配置写入 cfgPath
func initVault(cmd *cobra.Command, dir, cfgPath string, force bool, cfg *models.Config, maintain bool, sourceBranch string) error {
	vaultDir := vault.VaultRoot(dir)

	// 创建目录（维护者模式也建缓存目录；菜谱数据只存在于数据源检出，不落在代码分支）
	for _, d := range []string{
		vaultDir,
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

	// 维护者 + 同仓库数据分支：创建"数据仓库"分支 + worktree + 忽略配置
	if maintain && strings.TrimSpace(cfg.Source.Repo) == "" {
		if err := setupRecipesBranch(cmd, dir, sourceBranch, cfg); err != nil {
			return err
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔ Flavor Vault 初始化完成：%s\n", vaultDir)
	fmt.Fprintf(cmd.OutOrStdout(), "  配置: %s\n", cfgPath)
	if maintain {
		if strings.TrimSpace(cfg.Source.Repo) == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  维护菜谱：fv add / fv edit / fv rm，推送：fv gh push（数据源分支 %s）\n", sourceBranch)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  数据源：%s（分支 %s）。运行 fv source pull 检出后维护菜谱。\n", cfg.Source.Repo, sourceBranch)
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  使用者模式：fv list / fv search / fv show / fv stats 从 endpoint（或本地数据）读取。")
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
