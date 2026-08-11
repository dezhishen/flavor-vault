package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "rm <id>",
		Short: "删除菜谱（经 GitHub API 删除数据源分支文件；支持 --action-id 缓存删除意图）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			// 编辑目标：GitHub API（数据源分支）
			cfg, _, cfgPath, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			cl, branch, projectRoot, err := recipeAPIClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			st := actionStoreFor(cmd)

			// 先确认存在
			exists, err := cl.FileExists(ctx, branch, apiRecipePath(id))
			if err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("菜谱 %q 不存在", id)
			}

			// 缓存删除意图（供 AI/人追踪与续写）
			if st != nil {
				if err := cacheRecipe(st, "rm", id, nil); err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "⚠ 缓存删除意图失败: %v\n", err)
				} else {
					fmt.Fprintf(cmd.ErrOrStderr(), "ℹ 删除意图已缓存到 %s\n", st.Path())
				}
			}

			if !yes {
				confirmed, err := promptBool(newLineReader(), fmt.Sprintf("确认删除菜谱 %q?", id), false)
				if err != nil {
					return err
				}
				if !confirmed {
					fmt.Fprintln(cmd.OutOrStdout(), "已取消（缓存已保留，可用 fv action clear 清除）")
					return nil
				}
			}

			if err := apiDeleteRecipe(ctx, cl, branch, id, cfg, cfgPath, projectRoot, fmt.Sprintf("rm: %s", id)); err != nil {
				return err
			}
			completeAction(cmd, st)
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已删除菜谱 %s\n", id)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "跳过确认")
	return cmd
}
