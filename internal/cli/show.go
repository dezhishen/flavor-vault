package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"flavor-vault/internal/data"
	"flavor-vault/internal/store"
)

func newShowCmd() *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "打印完整菜谱（格式化 JSON，支持远程 endpoint）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}

			// 使用者模式：endpoint（或默认）非空 → 读取部署好的 details/<id>.json
			locator, remote := data.Locator(cfg, projectRoot, "details/"+args[0]+".json")
			if remote {
				rawData, err := data.ReadJSON(locator, remote)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(rawData))
				return nil
			}

			// 维护者模式：读取本地菜谱文件
			fs := store.NewRecipeFileStore(recipesDir(cfg, projectRoot))
			r, err := fs.Load(args[0])
			if err != nil {
				return err
			}
			var out []byte
			if raw {
				out, err = json.Marshal(r)
			} else {
				out, err = json.MarshalIndent(r, "", "  ")
			}
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
			return nil
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "紧凑单行输出")
	return cmd
}
