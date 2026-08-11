package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"flavor-vault/internal/pipeline"
	"flavor-vault/internal/plugins"
)

var version = "0.1.0"

// NewRootCommand 创建根命令
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:     "fv",
		Short:   "Flavor Vault - 基于 Git 的菜谱管理工具",
		Long:    "Flavor Vault 是一个基于 CLI 的菜谱管理工具，通过 Git + 纯静态托管实现无服务器部署。",
		Version: version,
	}

	// 全局 action-id 参数：菜谱维护命令将操作参数缓存到
	// /tmp/flavor-vaults/action-<id>.json，校验无误后完成动作，
	// 有误时保留缓存供 AI/人修正后以相同 action-id 重试。
	root.PersistentFlags().String("action-id", "",
		"操作 ID：将操作参数缓存到 /tmp/flavor-vaults/action-<id>.json，校验无误后完成动作")

	// 全局 --config/-c 参数：指定配置文件路径（默认 <root>/.flavor-vault/config.yaml）
	root.PersistentFlags().StringP("config", "c", "",
		"配置文件路径（默认 .flavor-vault/config.yaml）")

	root.AddCommand(
		newInitCmd(),
		newAddCmd(),
		newEditCmd(),
		newRmCmd(),
		newListCmd(),
		newShowCmd(),
		newBuildCmd(),
		newAskCmd(),
		newPushCmd(),
		newGhCmd(),
		newSourceCmd(),
		newConfigCmd(),
		newActionCmd(),
	)

	// 注册插件子命令（filter / stats）
	if err := registerPluginCommands(root); err != nil {
		fmt.Fprintln(os.Stderr, "注册插件命令失败:", err)
	}

	return root
}

// registerPluginCommands 实例化所有插件并注册其子命令
func registerPluginCommands(root *cobra.Command) error {
	sch := pipeline.NewScheduler(os.Stderr)
	sch.AddPlugin(&plugins.Validator{})
	sch.AddPlugin(&plugins.FacetIndexer{})
	sch.AddPlugin(&plugins.TagIndexer{})
	sch.AddPlugin(&plugins.DetailSplitter{})
	sch.AddPlugin(&plugins.StatsCollector{})
	sch.AddPlugin(&plugins.AIExporter{})
	sch.AddPlugin(&plugins.SearchIndexer{})
	return sch.RegisterCommands(root)
}
