package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"flavor-vault/internal/store"
)

func newShowCmd() *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "打印完整菜谱（格式化 JSON）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			fs := store.NewRecipeFileStore(recipesDir(cfg, projectRoot))
			r, err := fs.Load(args[0])
			if err != nil {
				return err
			}
			var data []byte
			if raw {
				data, err = json.Marshal(r)
			} else {
				data, err = json.MarshalIndent(r, "", "  ")
			}
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "紧凑单行输出")
	return cmd
}
