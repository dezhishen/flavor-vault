package cli

import (
	"fmt"
	"strings"

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
			mode := "使用者"
			if cfg.Maintainer() {
				mode = "维护者"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "配置文件: %s\n", cfgPath)
			fmt.Fprintf(cmd.OutOrStdout(), "模式: %s\n", mode)
			if cfg.Maintainer() {
				repo := orDash(cfg.Source.Repo)
				if strings.TrimSpace(cfg.Source.Repo) == "" {
					repo = "（当前仓库）"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "数据源: %s（分支 %s）\n", repo, orDash(cfg.Source.Branch))
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "数据源: 未配置（通过 endpoint 读取）")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "endpoint: %s\n", orDash(cfg.Endpoint))
			fmt.Fprintf(cmd.OutOrStdout(), "asset_dir: %s\n", orDash(cfg.AssetDir))
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "修改配置项（endpoint / asset_dir / source.repo / source.branch）",
		Example: `  fv config set endpoint https://user.github.io/flavor-vault/data
  fv config set asset_dir custom/assets
  fv config set source.repo owner/recipes
  fv config set source.branch recipes`,
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
			case "source.repo":
				cfg.Source.Repo = args[1]
			case "source.branch":
				cfg.Source.Branch = args[1]
			default:
				return fmt.Errorf("暂不支持配置项 %q（支持 endpoint / asset_dir / source.repo / source.branch）", args[0])
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
