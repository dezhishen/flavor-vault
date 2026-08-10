package plugins

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"flavor-vault/internal/pipeline"
)

// TagIndexer 按标签分组，生成每个标签下的菜谱列表（轻量）
type TagIndexer struct{}

// Name 插件标识
func (p *TagIndexer) Name() string { return "tag_indexer" }

// RegisterCommands tag_indexer 不注册子命令
func (p *TagIndexer) RegisterCommands(_ *cobra.Command) error { return nil }

// Build 生成 by-tag/{tag}.json
func (p *TagIndexer) Build(ctx *pipeline.BuildContext) error {
	outDir := filepath.Join(ctx.DataDir, "by-tag")
	return cachedWriteFiles(ctx, p.Name(), outDir, func() (map[string][]byte, error) {
		// 收集所有标签
		tagSet := make(map[string]bool)
		for _, r := range ctx.Recipes {
			for _, t := range r.Tags {
				tagSet[t] = true
			}
		}
		// 每组排序保持确定性
		groups := make(map[string][]RecipeSummary)
		for tag := range tagSet {
			groups[tag] = []RecipeSummary{}
		}
		for _, r := range ctx.Recipes {
			for _, t := range r.Tags {
				groups[t] = append(groups[t], ToSummary(r))
			}
		}

		files := make(map[string][]byte, len(groups))
		for tag, list := range groups {
			data, err := json.MarshalIndent(list, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("序列化标签 %q 失败: %w", tag, err)
			}
			files[tag+".json"] = data
		}
		return files, nil
	})
}
