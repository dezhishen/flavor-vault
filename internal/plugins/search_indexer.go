package plugins

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/data"
	"flavor-vault/internal/models"
	"flavor-vault/internal/pipeline"
	"flavor-vault/internal/vault"
)

// SearchEntry 搜索索引单条记录（dist/data/search.json）
// 前端与 CLI 共用同一份数据：字段拼合后做子串匹配
type SearchEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Kitchenware []string `json:"kitchenware"`
	Ingredients []string `json:"ingredients"` // 主要 + 配菜食材名
	Steps       string   `json:"steps"`       // 步骤描述（"1. ...\n2. ..."），用于全文检索
	Cover       string   `json:"cover,omitempty"`
	PrepTime    int      `json:"prep_time"`
	CookTime    int      `json:"cook_time"`
	Difficulty  int      `json:"difficulty"`
}

// SearchIndexer 生成搜索索引（前端与 CLI 搜索共用）
type SearchIndexer struct{}

// Name 插件标识
func (p *SearchIndexer) Name() string { return "search_indexer" }

// RegisterCommands 注册 fv search 子命令
func (p *SearchIndexer) RegisterCommands(root *cobra.Command) error {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "search <关键词>",
		Short: "搜索菜谱（本地 dist/data 或远程 endpoint，与前端同一套 search.json）",
		Example: `  fv search 红烧
  fv search 鸡翅 烤箱
  fv search "凉菜 快手" --json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			configFlag, _ := cmd.Flags().GetString("config")
			projectRoot, cfgPath, err := vault.ResolveContext(configFlag)
			if err != nil {
				return err
			}
			cfg, err := vault.LoadConfigAt(cfgPath)
			if err != nil {
				return err
			}
			locator, remote := data.Locator(cfg, projectRoot, "search.json")
			raw, err := data.ReadJSON(locator, remote)
			if err != nil {
				return err
			}
			var entries []SearchEntry
			if err := json.Unmarshal(raw, &entries); err != nil {
				return err
			}
			results := MatchSearch(entries, query)

			if jsonOut {
				out, _ := json.MarshalIndent(results, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(out))
				return nil
			}

			if len(results) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(未找到匹配菜谱，尝试更简单的关键词)")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "共 %d 个结果：\n", len(results))
			for _, e := range results {
				tags := ""
				if len(e.Tags) > 0 {
					tags = " [" + strings.Join(e.Tags, ",") + "]"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s（%s）%s %d分钟 ★%d\n",
					e.Name, e.ID, tags, e.PrepTime+e.CookTime, e.Difficulty)
				if e.Description != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", e.Description)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "以 JSON 数组输出结果")
	root.AddCommand(cmd)
	return nil
}

// Build 生成 search.json（每条菜谱一条可检索记录）
func (p *SearchIndexer) Build(ctx *pipeline.BuildContext) error {
	outPath := filepath.Join(ctx.DataDir, "search.json")
	return cachedWrite(ctx, p.Name(), outPath, func() ([]byte, error) {
		entries := make([]SearchEntry, 0, len(ctx.Recipes))
		for _, r := range ctx.Recipes {
			entries = append(entries, toSearchEntry(r))
		}
		return json.MarshalIndent(entries, "", "  ")
	})
}

// toSearchEntry 由菜谱生成搜索条目
func toSearchEntry(r *models.Recipe) SearchEntry {
	ings := r.MainIngredientNames()
	for _, s := range r.Ingredients.Side {
		ings = append(ings, s.Name)
	}
	var steps []string
	for _, s := range r.Steps {
		steps = append(steps, fmt.Sprintf("%d. %s", s.Order, s.Description))
	}
	return SearchEntry{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Tags:        r.Tags,
		Kitchenware: r.Kitchenware,
		Ingredients: ings,
		Steps:       strings.Join(steps, "\n"),
		Cover:       r.Media.Cover,
		PrepTime:    r.Stats.PrepTime,
		CookTime:    r.Stats.CookTime,
		Difficulty:  r.Stats.Difficulty,
	}
}

// haystack 返回该条目用于匹配的拼合文本（小写）
func (e *SearchEntry) haystack() string {
	parts := []string{e.Name, e.Description}
	parts = append(parts, e.Tags...)
	parts = append(parts, e.Kitchenware...)
	parts = append(parts, e.Ingredients...)
	if e.Steps != "" {
		parts = append(parts, e.Steps)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

// MatchSearch 关键词匹配：按空白拆分 token，全部命中（AND）才算匹配；大小写不敏感
func MatchSearch(entries []SearchEntry, query string) []SearchEntry {
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(tokens) == 0 {
		return nil
	}
	var out []SearchEntry
	for _, e := range entries {
		text := e.haystack()
		ok := true
		for _, t := range tokens {
			if !strings.Contains(text, t) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, e)
		}
	}
	return out
}
