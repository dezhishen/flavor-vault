package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"flavor-vault/internal/pipeline"
	"flavor-vault/internal/plugins"
	"flavor-vault/internal/store"
	"flavor-vault/internal/vault"
)

func newBuildCmd() *cobra.Command {
	var (
		force      bool
		incremental bool
		output     string
	)
	cmd := &cobra.Command{
		Use:   "build",
		Short: "执行 ETL 流水线，生成静态站点",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()

			cfg, projectRoot, cfgPath, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			if output != "" {
				cfg.OutputDir = output
			}
			outDir := vault.ResolveOutputDir(projectRoot, cfg)

			// 加载所有菜谱（格式错误文件跳过并警告，validator 强制校验）
			res, err := store.LoadAll(recipesDir(cfg, projectRoot), store.LoadOptions{SkipInvalid: true})
			if err != nil {
				return err
			}
			for _, w := range res.Warnings {
				fmt.Fprintln(cmd.ErrOrStderr(), "⚠", w)
			}

			ctx := pipeline.NewBuildContext(res.Recipes, cfg, outDir, vault.CacheRoot(projectRoot), cfgPath, force)
			ctx.AssetDir = vault.ResolveAssetDir(projectRoot, cfg)

			// 注册插件（按依赖顺序）
			scheduler := pipeline.NewScheduler(cmd.ErrOrStderr())
			scheduler.AddPlugin(&plugins.Validator{Logger: func(format string, a ...interface{}) {
				fmt.Fprintf(cmd.ErrOrStderr(), format, a...)
			}})
			scheduler.AddPlugin(&plugins.FacetIndexer{})
			scheduler.AddPlugin(&plugins.TagIndexer{})
			scheduler.AddPlugin(&plugins.DetailSplitter{})
			scheduler.AddPlugin(&plugins.AssetCollector{})
			scheduler.AddPlugin(&plugins.StatsCollector{})
			scheduler.AddPlugin(&plugins.AIExporter{})

			if err := scheduler.Run(ctx); err != nil {
				return err
			}

			// 复制前端静态资源（若 web/dist 存在）
			copied := copyFrontendAssets(projectRoot, outDir, cmd.ErrOrStderr())

			fmt.Fprintf(cmd.OutOrStdout(), "\n✔ 构建完成: %d 道菜谱 -> %s（耗时 %s）\n",
				len(res.Recipes), outDir, time.Since(start).Round(time.Millisecond))
			if copied {
				fmt.Fprintf(cmd.OutOrStdout(), "✔ 已合并前端静态资源\n")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "ℹ 未找到前端构建产物（web/dist），请先在前端目录运行 pnpm build\n")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "强制重建（忽略缓存）")
	cmd.Flags().BoolVar(&incremental, "incremental", false, "增量构建（基于缓存自动判断，默认开启）")
	cmd.Flags().StringVar(&output, "output", "", "覆盖输出目录")
	return cmd
}

// copyFrontendAssets 将 web/dist 内容复制到输出目录根
func copyFrontendAssets(projectRoot, outDir string, log io.Writer) bool {
	webDist := filepath.Join(projectRoot, "web", "dist")
	info, err := os.Stat(webDist)
	if err != nil || !info.IsDir() {
		return false
	}
	count := copyDir(webDist, outDir, log)
	fmt.Fprintf(log, "✔ 复制前端资源 %d 个文件\n", count)
	return true
}

func copyDir(src, dst string, log io.Writer) int {
	count := 0
	_ = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil
		}
		if err := os.WriteFile(target, data, info.Mode()); err == nil {
			count++
		}
		return nil
	})
	return count
}
