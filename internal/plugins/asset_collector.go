package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"flavor-vault/internal/pipeline"
	"flavor-vault/internal/utils"
)

// AssetCollector 收集菜谱引用的本地图片/资源，复制到 dist/assets/<ref>。
// 外部 URL（http/https/data:）直接透传，不参与复制。
type AssetCollector struct{}

// Name 插件标识
func (p *AssetCollector) Name() string { return "asset_collector" }

// RegisterCommands asset_collector 不注册子命令
func (p *AssetCollector) RegisterCommands(_ *cobra.Command) error { return nil }

// Build 将菜谱引用的资源复制到输出 assets 目录
func (p *AssetCollector) Build(ctx *pipeline.BuildContext) error {
	outDir := filepath.Join(ctx.OutputDir, "assets")
	return cachedWriteFiles(ctx, p.Name(), outDir, func() (map[string][]byte, error) {
		files := make(map[string][]byte)
		seen := make(map[string]bool)
		for _, r := range ctx.Recipes {
			for _, ref := range r.AssetRefs() {
				if seen[ref] || utils.IsRemoteURL(ref) {
					continue
				}
				seen[ref] = true
				src := filepath.Join(ctx.AssetDir, filepath.FromSlash(ref))
				data, err := os.ReadFile(src)
				if err != nil {
					fmt.Fprintf(os.Stderr, "⚠ 资源缺失: %s（菜谱 %s）\n", src, r.ID)
					continue
				}
				files[filepath.ToSlash(ref)] = data
			}
		}
		return files, nil
	})
}
