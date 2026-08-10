package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"flavor-vault/internal/action"
)

// newActionCmd 管理基于 action-id 的操作缓存
func newActionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "action",
		Short: "管理基于 action-id 的操作缓存（/tmp/flavor-vaults）",
	}

	cmd.AddCommand(
		newActionListCmd(),
		newActionShowCmd(),
		newActionClearCmd(),
	)
	return cmd
}

func newActionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出所有已缓存的操作",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			actions, err := action.List()
			if err != nil {
				return err
			}
			if len(actions) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(无缓存操作)")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "共 %d 个待处理操作（目录: %s）:\n", len(actions), action.Dir())
			for _, a := range actions {
				target := a.TargetID
				if target == "" {
					target = "-"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %-16s %-6s -> %-24s 更新于 %s\n",
					a.ActionID, a.Action, target, a.UpdatedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
}

func newActionShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <action-id>",
		Short: "显示已缓存操作的参数（JSON）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := action.New(args[0])
			if !st.Exists() {
				return fmt.Errorf("未找到操作缓存 %s", st.Path())
			}
			a, err := st.Load()
			if err != nil {
				return err
			}
			data, err := json.MarshalIndent(a, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}

func newActionClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear <action-id>",
		Short: "清除某个已缓存的操作",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st := action.New(args[0])
			if !st.Exists() {
				return fmt.Errorf("未找到操作缓存 %s", st.Path())
			}
			if err := st.Clear(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已清除缓存 %s\n", st.Path())
			return nil
		},
	}
}
