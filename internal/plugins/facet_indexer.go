package plugins

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/data"
	"flavor-vault/internal/models"
	"flavor-vault/internal/pipeline"
	"flavor-vault/internal/utils"
	"flavor-vault/internal/vault"
)

// FacetIndex 倒排索引：厨具/主要食材/标签
type FacetIndex struct {
	Kitchenware map[string][]string `json:"kitchenware"` // 厨具 -> 有序 ID 列表
	Ingredients map[string][]string `json:"ingredients"` // 主要食材 -> 有序 ID 列表
	Tags        map[string][]string `json:"tags"`        // 标签 -> 有序 ID 列表
}

// NewFacetIndex 创建空索引
func NewFacetIndex() *FacetIndex {
	return &FacetIndex{
		Kitchenware: make(map[string][]string),
		Ingredients: make(map[string][]string),
		Tags:        make(map[string][]string),
	}
}

// FacetIndexer 构建厨具/主要食材/标签倒排索引
type FacetIndexer struct{}

// Name 插件标识
func (p *FacetIndexer) Name() string { return "facet_indexer" }

// RegisterCommands 注册 fv filter 子命令
func (p *FacetIndexer) RegisterCommands(root *cobra.Command) error {
	var (
		kitchenware []string
		tags        []string
		ingredients []string
		jsonOut     bool
	)
	cmd := &cobra.Command{
		Use:   "filter",
		Short: "基于倒排索引求交集，返回匹配的菜谱 ID",
		Example: `  fv filter --厨具 炒锅 --标签 凉菜
  fv filter --食材 鸡翅 --厨具 烤箱 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configFlag, _ := cmd.Flags().GetString("config")
			projectRoot, cfgPath, err := vault.ResolveContext(configFlag)
			if err != nil {
				return err
			}
			cfg, err := vault.LoadConfigAt(cfgPath)
			if err != nil {
				return err
			}
			// 支持远程 endpoint（与 pages 同一套数据）或本地 dist/data
			locator, remote := data.Locator(cfg, projectRoot, "filters.json")
			raw, err := data.ReadJSON(locator, remote)
			if err != nil {
				return err
			}
			var idx FacetIndex
			if err := json.Unmarshal(raw, &idx); err != nil {
				return err
			}
			return runFilter(&idx, kitchenware, tags, ingredients, jsonOut, cmd)
		},
	}
	cmd.Flags().StringSliceVar(&kitchenware, "厨具", nil, "厨具过滤（可多次指定或逗号分隔）")
	cmd.Flags().StringSliceVar(&tags, "标签", nil, "标签过滤（可多次指定或逗号分隔）")
	cmd.Flags().StringSliceVar(&ingredients, "食材", nil, "主要食材过滤（可多次指定或逗号分隔）")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "以 JSON 数组输出")
	root.AddCommand(cmd)
	return nil
}

// Build 构建倒排索引并输出到 filters.json
func (p *FacetIndexer) Build(ctx *pipeline.BuildContext) error {
	outPath := filepath.Join(ctx.DataDir, "filters.json")
	return cachedWrite(ctx, p.Name(), outPath, func() ([]byte, error) {
		idx := buildFacetIndex(ctx.Recipes)
		return json.MarshalIndent(idx, "", "  ")
	})
}

// buildFacetIndex 构建索引
func buildFacetIndex(recipes []*models.Recipe) *FacetIndex {
	idx := NewFacetIndex()
	for _, r := range recipes {
		for _, kw := range r.Kitchenware {
			idx.Kitchenware[kw] = append(idx.Kitchenware[kw], r.ID)
		}
		for _, ing := range r.MainIngredientNames() {
			idx.Ingredients[ing] = append(idx.Ingredients[ing], r.ID)
		}
		for _, t := range r.Tags {
			idx.Tags[t] = append(idx.Tags[t], r.ID)
		}
	}
	// 排序以保证交集算法正确
	for k := range idx.Kitchenware {
		sort.Strings(idx.Kitchenware[k])
	}
	for k := range idx.Ingredients {
		sort.Strings(idx.Ingredients[k])
	}
	for k := range idx.Tags {
		sort.Strings(idx.Tags[k])
	}
	return idx
}

// runFilter 求交集并输出
func runFilter(idx *FacetIndex, kitchenware, tags, ingredients []string, jsonOut bool, cmd *cobra.Command) error {
	lists := make([][]string, 0, len(kitchenware)+len(tags)+len(ingredients))
	hasFilter := false
	collect := func(keys []string, m map[string][]string, label string) error {
		for _, k := range keys {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			hasFilter = true
			ids, ok := m[k]
			if !ok {
				// 该条件无任何匹配
				ids = []string{}
			}
			lists = append(lists, ids)
		}
		return nil
	}
	if err := collect(kitchenware, idx.Kitchenware, "厨具"); err != nil {
		return err
	}
	if err := collect(tags, idx.Tags, "标签"); err != nil {
		return err
	}
	if err := collect(ingredients, idx.Ingredients, "食材"); err != nil {
		return err
	}

	var result []string
	if !hasFilter {
		// 无过滤条件：输出所有 ID
		seen := make(map[string]bool)
		for _, ids := range idx.Kitchenware {
			for _, id := range ids {
				seen[id] = true
			}
		}
		for id := range seen {
			result = append(result, id)
		}
		sort.Strings(result)
	} else {
		result = utils.Intersect(lists...)
	}

	if jsonOut {
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}
	if len(result) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "(无匹配菜谱)")
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), strings.Join(result, "\n"))
	return nil
}
