package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
	"flavor-vault/internal/plugins"
)

func newEditCmd() *cobra.Command {
	var jsonInput string
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "编辑已有菜谱（$EDITOR 或 --json 补丁；经 GitHub API 更新数据源分支文件）",
		Args:  cobra.ExactArgs(1),
		Example: `  fv edit hong-shao-rou                               # $EDITOR 编辑
  fv edit hong-shao-rou --json '{"stats":{"difficulty":4}}'   # 局部更新
  fv edit hong-shao-rou --action-id e1 --json @patch.json     # 缓存补丁，失败可续写`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, cfgPath, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			id := args[0]

			// 编辑目标：GitHub API（数据源分支）
			cl, branch, projectRoot, err := recipeAPIClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			st := actionStoreFor(cmd)

			// 1. 从数据源分支读取目标菜谱
			base, err := apiLoadRecipe(ctx, cl, branch, id)
			if err != nil {
				return err
			}

			// 2. 从缓存恢复（若存在该 action-id 的 edit 缓存）
			var restored bool
			if cached, ok := loadCachedRecipe(st, "edit"); ok && cached != nil && cached.ID == id {
				base = cached
				restored = true
				fmt.Fprintf(cmd.ErrOrStderr(), "ℹ 已从缓存恢复编辑草稿（action-id=%s）\n", st.ID)
			}

			// 3. 应用 --json 补丁（未提供的字段保持原值）
			if jsonInput != "" {
				if err := parseRecipeJSON(jsonInput, base); err != nil {
					return err
				}
			} else if !restored {
				// 4. 无补丁且无缓存 → $EDITOR 交互编辑（临时文件）
				tmp, err := os.CreateTemp("", "fv-recipe-*.json")
				if err != nil {
					return err
				}
				tmpPath := tmp.Name()
				raw, err := json.MarshalIndent(base, "", "  ")
				if err == nil {
					_, err = tmp.Write(raw)
				}
				_ = tmp.Close()
				if err != nil {
					return err
				}
				defer os.Remove(tmpPath)

				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi"
				}
				e := exec.Command(editor, tmpPath)
				e.Stdin = os.Stdin
				e.Stdout = os.Stdout
				e.Stderr = os.Stderr
				if err := e.Run(); err != nil {
					return fmt.Errorf("启动编辑器失败: %w", err)
				}
				// 重新读取编辑后的内容
				edited, err := os.ReadFile(tmpPath)
				if err != nil {
					return err
				}
				base = &models.Recipe{}
				if err := json.Unmarshal(edited, base); err != nil {
					return fmt.Errorf("编辑后的菜谱 JSON 解析失败: %w", err)
				}
				if st != nil {
					if err := cacheRecipe(st, "edit", id, base); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "⚠ 缓存编辑草稿失败: %v\n", err)
					} else {
						fmt.Fprintf(cmd.ErrOrStderr(), "ℹ 编辑内容已缓存到 %s\n", st.Path())
					}
				}
			}

			// 5. 强制保持目标 ID 与时间戳
			base.ID = id
			base.UpdatedAt = time.Now()

			// 6. 校验；有误则缓存补丁供重试
			if err := plugins.ValidateRecipe(base, cfg); err != nil {
				return failAndCache(cmd, st, "edit", id, base, err)
			}

			// 7. 经 GitHub API 提交更新（单文件）
			assetBase := cfg.AssetDir
			if assetBase == "" {
				assetBase = ".flavor-vault/assets"
			}
			if err := apiSaveRecipe(ctx, cl, branch, base, assetBase, assetDirFor(cfg, projectRoot), cfg, cfgPath, projectRoot,
				fmt.Sprintf("edit: %s", base.Name)); err != nil {
				return failAndCache(cmd, st, "edit", id, base, err)
			}
			completeAction(cmd, st)
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已更新并提交菜谱 %s (%s)\n", id, base.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&jsonInput, "json", "", "以 JSON 补丁方式更新字段（支持 @文件路径），未提供的字段保持不变")
	return cmd
}
