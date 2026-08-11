package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/data"
	"flavor-vault/internal/plugins"
)

func newAskCmd() *cobra.Command {
	var (
		top    int
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:   "ask <query>",
		Short: "AI 智能查询（自然语言），输出匹配菜谱（本地缓存优先）",
		Args:  cobra.MinimumNArgs(1),
		Example: `  fv ask "不用炒锅的凉菜"
  fv ask "烤箱能做的甜点" --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			// 支持远程 endpoint（与 pages 同一套数据）或本地 dist/data
			locator, remote := data.Locator(cfg, projectRoot, "ai-corpus.json")
			raw, err := data.ReadJSON(locator, remote)
			if err != nil {
				return err
			}
			entries, err := parseCorpus(raw)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(AI 语料为空，请先 fv build 或检查 endpoint)")
				return nil
			}

			keywords, negKeywords := extractKeywords(query)
			results := searchCorpus(entries, keywords, negKeywords)

			if jsonOut {
				data, _ := json.MarshalIndent(results, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}

			if len(results) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(未找到匹配菜谱，尝试更简单的关键词)")
				return nil
			}
			limit := top
			if limit <= 0 || limit > len(results) {
				limit = len(results)
			}
			for i, r := range results[:limit] {
				fmt.Fprintf(cmd.OutOrStdout(), "%d. %s（%s） [%s] 评分 %d\n",
					i+1, r.Name, r.ID, strings.Join(r.Tags, ","), r.Score)
				if r.Description != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", r.Description)
				}
				if len(r.MainIngredients) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "   主要食材: %s\n", strings.Join(r.MainIngredients, ", "))
				}
				fmt.Fprintf(cmd.OutOrStdout(), "   耗时: 准备%d分钟 烹饪%d分钟 | 难度: ★%d\n",
					r.PrepTime, r.CookTime, r.Difficulty)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&top, "top", 5, "返回前 N 条结果")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "以 JSON 输出")
	return cmd
}

// AskResult 检索结果
type AskResult struct {
	plugins.AICorpusEntry
	Score int `json:"score"`
}

// parseCorpus 解析 JSON Lines 格式的 AI 语料（来自本地文件或远程 endpoint）
func parseCorpus(raw []byte) ([]plugins.AICorpusEntry, error) {
	var entries []plugins.AICorpusEntry
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e plugins.AICorpusEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // 跳过损坏行
		}
		entries = append(entries, e)
	}
	return entries, sc.Err()
}

// extractKeywords 从自然语言查询中提取正向与否定关键词
func extractKeywords(query string) ([]string, []string) {
	// 停用词
	stopwords := map[string]bool{
		"我": true, "想": true, "做": true, "一道": true, "个": true, "的": true,
		"要": true, "有": true, "用": true, "可以": true, "什么": true,
		"推荐": true, "给": true, "请问": true, "一点": true, "一些": true,
	}
	// 连接助词/分隔词：用于切分中文短语
	connectors := "的了和与或及和或者还有等以及跟同"

	var positive, negative []string
	for _, part := range strings.FieldsFunc(query, func(r rune) bool {
		return r == ' ' || r == ',' || r == '，' || r == '。' || r == '？' || r == '?' ||
			r == '!' || r == '！' || r == '、' || r == '；' || r == ';'
	}) {
		// 进一步按连接词切分
		subs := splitByConnectors(part, connectors)
		for _, sub := range subs {
			sub = strings.TrimSpace(sub)
			if sub == "" || stopwords[sub] {
				continue
			}
			// 否定前缀
			neg := false
			for _, p := range []string{"不用", "不要", "没有", "无需", "无"} {
				if strings.HasPrefix(sub, p) {
					neg = true
					sub = strings.TrimPrefix(sub, p)
					break
				}
			}
			sub = strings.TrimSpace(sub)
			if sub == "" {
				continue
			}
			if neg {
				negative = append(negative, sub)
			} else {
				positive = append(positive, sub)
			}
		}
	}
	return positive, negative
}

// splitByConnectors 按中文字符连接词切分
func splitByConnectors(s, connectors string) []string {
	var out []string
	var buf []rune
	flush := func() {
		if len(buf) > 0 {
			out = append(out, string(buf))
			buf = nil
		}
	}
	for _, r := range s {
		if strings.ContainsRune(connectors, r) {
			flush()
		} else {
			buf = append(buf, r)
		}
	}
	flush()
	return out
}

// matchesKeyword 判断条目是否命中某关键词
func matchesKeyword(e plugins.AICorpusEntry, kw string) bool {
	if e.Name == kw || strings.Contains(e.Name, kw) || strings.Contains(kw, e.Name) {
		return true
	}
	for _, t := range e.Tags {
		if t == kw {
			return true
		}
	}
	for _, ing := range e.MainIngredients {
		if ing == kw || strings.Contains(ing, kw) {
			return true
		}
	}
	for _, kw2 := range e.Kitchenware {
		if kw2 == kw {
			return true
		}
	}
	if e.Description != "" && strings.Contains(e.Description, kw) {
		return true
	}
	return false
}

// searchCorpus 基于关键词打分检索（支持否定词）
func searchCorpus(entries []plugins.AICorpusEntry, keywords, negKeywords []string) []AskResult {
	var results []AskResult
	for _, e := range entries {
		// 否定词：若命中则排除
		excluded := false
		for _, kw := range negKeywords {
			if matchesKeyword(e, kw) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		score := 0
		for _, kw := range keywords {
			if e.Name == kw || strings.Contains(e.Name, kw) || strings.Contains(kw, e.Name) {
				score += 10
			}
			for _, t := range e.Tags {
				if t == kw {
					score += 8
				}
			}
			for _, ing := range e.MainIngredients {
				if ing == kw || strings.Contains(ing, kw) {
					score += 6
				}
			}
			for _, kw2 := range e.Kitchenware {
				if kw2 == kw {
					score += 6
				}
			}
			if e.Description != "" && strings.Contains(e.Description, kw) {
				score += 3
			}
		}
		if score > 0 {
			results = append(results, AskResult{AICorpusEntry: e, Score: score})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].ID < results[j].ID
	})
	return results
}
