package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"flavor-vault/internal/vault"
)

// newConfigCmd 查看/修改配置（使用者改 endpoint 即可切换数据源）
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "查看/修改配置（如远程数据 endpoint）",
	}
	cmd.AddCommand(newConfigGetCmd(), newConfigSetCmd())
	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "显示当前配置摘要",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, cfgPath, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "配置文件: %s\n", cfgPath)
			fmt.Fprintf(cmd.OutOrStdout(), "endpoint: %s\n", orDash(cfg.Endpoint))
			fmt.Fprintf(cmd.OutOrStdout(), "asset_dir: %s\n", orDash(cfg.AssetDir))
			fmt.Fprintf(cmd.OutOrStdout(), "recipes_branch: %s\n", orDash(cfg.GitHub.RecipesBranch))
			fmt.Fprintf(cmd.OutOrStdout(), "外部数据源: %d 个\n", len(cfg.Sources))
			for _, s := range cfg.Sources {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %s -> %s (分支 %s)\n", s.Name, s.Repo, s.Branch)
			}
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "修改配置项（endpoint / asset_dir）",
		Example: `  fv config set endpoint https://user.github.io/flavor-vault/data
  fv config set asset_dir custom/assets`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, cfgPath, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			switch args[0] {
			case "endpoint":
				cfg.Endpoint = args[1]
			case "asset_dir":
				cfg.AssetDir = args[1]
			default:
				return fmt.Errorf("暂不支持配置项 %q（支持 endpoint / asset_dir）", args[0])
			}
			if err := vault.SaveConfigAt(cfgPath, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已设置 %s = %s\n", args[0], args[1])
			return nil
		},
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
