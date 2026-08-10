package plugins

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"flavor-vault/internal/pipeline"
)

// DetailSplitter 将完整 Steps 拆分到独立文件，按 ID 命名
type DetailSplitter struct{}

// Name 插件标识
func (p *DetailSplitter) Name() string { return "detail_splitter" }

// RegisterCommands detail_splitter 不注册子命令
func (p *DetailSplitter) RegisterCommands(_ *cobra.Command) error { return nil }

// Build 生成 details/{id}.json（完整菜谱数据）
func (p *DetailSplitter) Build(ctx *pipeline.BuildContext) error {
	outDir := filepath.Join(ctx.DataDir, "details")
	return cachedWriteFiles(ctx, p.Name(), outDir, func() (map[string][]byte, error) {
		files := make(map[string][]byte, len(ctx.Recipes))
		for _, r := range ctx.Recipes {
			data, err := json.MarshalIndent(r, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("序列化菜谱 %s 失败: %w", r.ID, err)
			}
			files[r.ID+".json"] = data
		}
		return files, nil
	})
}
