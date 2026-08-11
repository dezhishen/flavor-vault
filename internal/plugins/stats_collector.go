package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"flavor-vault/internal/data"
	"flavor-vault/internal/models"
	"flavor-vault/internal/pipeline"
	"flavor-vault/internal/vault"
)

// Meta 统计信息（dist/data/meta.json）
type Meta struct {
	Total        int            `json:"total"`
	GeneratedAt  time.Time      `json:"generated_at"`
	Tags         map[string]int `json:"tags"`         // 标签分布
	Kitchenware  map[string]int `json:"kitchenware"`  // 常用厨具
	Difficulty   map[string]int `json:"difficulty"`   // 难度分布
	AvgPrepTime  int            `json:"avg_prep_time"`
	AvgCookTime  int            `json:"avg_cook_time"`
	AvgTotalTime int            `json:"avg_total_time"`
	// 全部轻量列表（all.json 与 meta.json 分开存储）
	AllCount int `json:"all_count"`
	// 默认数据 endpoint（构建时注入，消费端未配置时使用；构建时可替换）
	Endpoint string `json:"endpoint,omitempty"`
}

// StatsCollector 统计总数、常用厨具、难度分布等
type StatsCollector struct{}

// Name 插件标识
func (p *StatsCollector) Name() string { return "stats_collector" }

// RegisterCommands 注册 fv stats 子命令
func (p *StatsCollector) RegisterCommands(root *cobra.Command) error {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "显示菜谱统计信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 从 dist 或远程 endpoint 加载 meta.json
			configFlag, _ := cmd.Flags().GetString("config")
			projectRoot, cfgPath, err := vault.ResolveContext(configFlag)
			if err != nil {
				return err
			}
			cfg, err := vault.LoadConfigAt(cfgPath)
			if err != nil {
				return err
			}
			locator, remote := data.Locator(cfg, projectRoot, "meta.json")
			raw, err := data.ReadJSON(locator, remote)
			if err != nil {
				return err
			}
			var m Meta
			if err := json.Unmarshal(raw, &m); err != nil {
				return err
			}
			meta := &m
			if jsonOut {
				data, _ := json.MarshalIndent(meta, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			printMeta(meta, cmd)
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "以 JSON 输出")
	root.AddCommand(cmd)
	return nil
}

// Build 生成 meta.json 与 all.json
func (p *StatsCollector) Build(ctx *pipeline.BuildContext) error {
	outDir := ctx.DataDir
	// 构建时可注入默认 endpoint（--endpoint / FV_ENDPOINT），写入 meta.json 供消费端未配置时使用
	injected := ""
	if v, ok := ctx.Options["endpoint"]; ok {
		if s, ok := v.(string); ok {
			injected = strings.TrimSpace(s)
		}
	}
	if injected == "" {
		injected = strings.TrimSpace(os.Getenv("FV_ENDPOINT"))
	}
	return cachedWriteFiles(ctx, p.Name(), outDir, func() (map[string][]byte, error) {
		meta := buildMeta(ctx.Recipes)
		if injected != "" {
			meta.Endpoint = injected
		}
		metaData, err := json.MarshalIndent(meta, "", "  ")
		if err != nil {
			return nil, err
		}
		allData, err := json.MarshalIndent(Summaries(ctx.Recipes), "", "  ")
		if err != nil {
			return nil, err
		}
		return map[string][]byte{
			"meta.json": metaData,
			"all.json":  allData,
		}, nil
	})
}

// buildMeta 计算统计信息
func buildMeta(recipes []*models.Recipe) *Meta {
	m := &Meta{
		Total:       len(recipes),
		GeneratedAt: time.Now(),
		Tags:        make(map[string]int),
		Kitchenware: make(map[string]int),
		Difficulty:  make(map[string]int),
	}
	var prepSum, cookSum int
	for _, r := range recipes {
		for _, t := range r.Tags {
			m.Tags[t]++
		}
		for _, kw := range r.Kitchenware {
			m.Kitchenware[kw]++
		}
		m.Difficulty[fmt.Sprintf("%d", r.Stats.Difficulty)]++
		prepSum += r.Stats.PrepTime
		cookSum += r.Stats.CookTime
	}
	if m.Total > 0 {
		m.AvgPrepTime = prepSum / m.Total
		m.AvgCookTime = cookSum / m.Total
		m.AvgTotalTime = (prepSum + cookSum) / m.Total
	}
	m.AllCount = m.Total
	return m
}

func printMeta(m *Meta, cmd *cobra.Command) {
	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "菜谱总数: %d\n", m.Total)
	fmt.Fprintf(w, "生成时间: %s\n", m.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "平均耗时: 准备 %d 分钟 / 烹饪 %d 分钟（合计 %d 分钟）\n", m.AvgPrepTime, m.AvgCookTime, m.AvgTotalTime)
	fmt.Fprintln(w, "难度分布:")
	for _, d := range []string{"1", "2", "3", "4", "5"} {
		if n, ok := m.Difficulty[d]; ok && n > 0 {
			fmt.Fprintf(w, "  ★%s: %d\n", d, n)
		}
	}
	fmt.Fprintln(w, "常用厨具 Top5:")
	topKitchenware(m.Kitchenware, 5, w)
	fmt.Fprintln(w, "标签分布 Top5:")
	topTags(m.Tags, 5, w)
}

func topKitchenware(m map[string]int, n int, w interface{ Write([]byte) (int, error) }) {
	type kv struct {
		k string
		v int
	}
	items := make([]kv, 0, len(m))
	for k, v := range m {
		items = append(items, kv{k, v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].v > items[j].v })
	for i := 0; i < len(items) && i < n; i++ {
		fmt.Fprintf(w, "  %s: %d\n", items[i].k, items[i].v)
	}
}

func topTags(m map[string]int, n int, w interface{ Write([]byte) (int, error) }) {
	topKitchenware(m, n, w)
}
