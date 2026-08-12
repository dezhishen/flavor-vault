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
	var yes bool
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

			// 3. 应用 --json 补丁（未提供的字段保持原值；多版本菜谱默认编辑第一个版本）
			if jsonInput != "" {
				if err := applyEditPatch(jsonInput, base); err != nil {
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

			// 暂存本地图片资源（编辑新增的本地图片复制到资源目录并随单文件上传）
			if n, err := stageLocalAssets(cfg, projectRoot, base); err != nil {
				return failAndCache(cmd, st, "edit", id, base, err)
			} else if n > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "ℹ 已暂存 %d 个本地图片\n", n)
			}

			// 7. 经 GitHub API 提交更新（单文件）
			assetBase := cfg.AssetDir
			if assetBase == "" {
				assetBase = ".flavor-vault/assets"
			}
			// 预览并确认提交（--yes 跳过；取消则保留编辑草稿供续写）
			ok, err := confirmCommit(cmd, base, fmt.Sprintf("edit: %s", base.Name))
			if err != nil {
				return err
			}
			if !ok {
				if st != nil {
					if cErr := cacheRecipe(st, "edit", id, base); cErr == nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "ℹ 已取消，编辑草稿保留在 %s\n", st.Path())
					}
				}
				return nil
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
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "跳过提交确认")
	return cmd
}

// applyEditPatch 应用 --json 编辑补丁：
//   - 补丁含 versions → 整体替换版本列表；
//   - 否则按版本字段（ingredients/seasonings/steps/media/stats）合并进第一个版本，顶层 name/description/tags/kitchenware 亦支持；
//   - 旧结构（无 versions）自动转为单版本结构，使编辑后也走多版本模型。
func applyEditPatch(raw string, r *models.Recipe) error {
	data, err := readJSONInput(raw)
	if err != nil {
		return err
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("解析菜谱 JSON 失败: %w", err)
	}

	// 补丁显式含 versions → 整体替换版本（菜谱变为多版本，顶层内容清空、内容在 versions）
	if rawVersions, ok := m["versions"]; ok {
		var vs []models.Version
		if err := json.Unmarshal(rawVersions, &vs); err != nil {
			return err
		}
		r.Versions = vs
		r.Ingredients = models.Ingredients{}
		r.Seasonings = nil
		r.Steps = nil
		r.Media = models.Media{}
		r.Stats = models.Stats{}
		return nil
	}

	// 顶层元数据字段
	if rawName, ok := m["name"]; ok {
		json.Unmarshal(rawName, &r.Name)
	}
	if rawDesc, ok := m["description"]; ok {
		json.Unmarshal(rawDesc, &r.Description)
	}
	if rawTags, ok := m["tags"]; ok {
		json.Unmarshal(rawTags, &r.Tags)
	}
	if rawKw, ok := m["kitchenware"]; ok {
		json.Unmarshal(rawKw, &r.Kitchenware)
	}

	apply := func(key string, dst interface{}) error {
		if rawVal, ok := m[key]; ok {
			if err := json.Unmarshal(rawVal, dst); err != nil {
				return fmt.Errorf("字段 %s 解析失败: %w", key, err)
			}
		}
		return nil
	}

	// 版本内容字段：统一合并进第一个版本；单版本菜谱（无 versions）先迁移为多版本（顶层入 versions[0]）
	normalizeMultiVersion(r)
	v := &r.Versions[0]
	if err := apply("ingredients", &v.Ingredients); err != nil {
		return err
	}
	if err := apply("seasonings", &v.Seasonings); err != nil {
		return err
	}
	if err := apply("steps", &v.Steps); err != nil {
		return err
	}
	if err := apply("media", &v.Media); err != nil {
		return err
	}
	if err := apply("stats", &v.Stats); err != nil {
		return err
	}
	if rawVName, ok := m["version"]; ok {
		json.Unmarshal(rawVName, &v.Name)
	}
	return nil
}
