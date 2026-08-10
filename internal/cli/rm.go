package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"flavor-vault/internal/store"
	"flavor-vault/internal/vault"
)

func newRmCmd() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "rm <id>",
		Short: "删除菜谱 JSON（支持 --action-id 缓存删除意图）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, _, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			id := args[0]
			fs := store.NewRecipeFileStore(vault.RecipesDir(projectRoot))
			if !fs.Exists(id) {
				return fmt.Errorf("菜谱 %q 不存在", id)
			}

			st := actionStoreFor(cmd)

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

			if err := fs.Delete(id); err != nil {
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
