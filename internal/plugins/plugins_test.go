package plugins

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"flavor-vault/internal/models"
	"flavor-vault/internal/pipeline"
	"flavor-vault/internal/utils"
)

func testRecipes() []*models.Recipe {
	return []*models.Recipe{
		{
			ID:   "r1",
			Name: "红烧肉",
			Tags: []string{"家常", "热菜"},
			Kitchenware: []string{"炒锅"},
			Ingredients: models.Ingredients{Main: []models.Ingredient{{Name: "五花肉"}}},
			Stats:       models.Stats{PrepTime: 20, CookTime: 70, Difficulty: 3},
		},
		{
			ID:   "r2",
			Name: "拍黄瓜",
			Tags: []string{"凉菜", "快手"},
			Kitchenware: []string{"保鲜袋"},
			Ingredients: models.Ingredients{Main: []models.Ingredient{{Name: "黄瓜"}}},
			Stats:       models.Stats{PrepTime: 10, CookTime: 0, Difficulty: 1},
		},
		{
			ID:   "r3",
			Name: "麻婆豆腐",
			Tags: []string{"川菜", "热菜"},
			Kitchenware: []string{"炒锅"},
			Ingredients: models.Ingredients{Main: []models.Ingredient{{Name: "嫩豆腐"}}},
			Stats:       models.Stats{PrepTime: 10, CookTime: 15, Difficulty: 2},
		},
	}
}

func newTestContext(t *testing.T, recipes []*models.Recipe, force bool) *pipeline.BuildContext {
	t.Helper()
	root := t.TempDir()
	outDir := filepath.Join(root, "dist")
	cacheRoot := filepath.Join(root, ".flavor-vault", "cache")
	return pipeline.NewBuildContext(recipes, models.DefaultConfig(), outDir, cacheRoot, filepath.Join(root, "config.yaml"), force)
}

func TestBuildFacetIndex(t *testing.T) {
	idx := buildFacetIndex(testRecipes())
	if !utils.Contains(idx.Kitchenware["炒锅"], "r1") {
		t.Error("r1 should be indexed under 炒锅")
	}
	if !utils.Contains(idx.Kitchenware["炒锅"], "r3") {
		t.Error("r3 should be indexed under 炒锅")
	}
	if !utils.Contains(idx.Tags["凉菜"], "r2") {
		t.Error("r2 should be indexed under 凉菜")
	}
	if !utils.Contains(idx.Ingredients["五花肉"], "r1") {
		t.Error("r1 should be indexed under 五花肉")
	}
	// 索引值必须有序（供交集算法使用）
	if !sort.StringsAreSorted(idx.Kitchenware["炒锅"]) {
		t.Error("index lists must be sorted")
	}
	// 交集：炒锅 ∩ 热菜 = r1, r3
	got := utils.Intersect(idx.Kitchenware["炒锅"], idx.Tags["热菜"])
	if len(got) != 2 || got[0] != "r1" || got[1] != "r3" {
		t.Errorf("intersect result = %v, want [r1 r3]", got)
	}
}

func TestFacetIndexerBuild(t *testing.T) {
	ctx := newTestContext(t, testRecipes(), true)
	p := &FacetIndexer{}
	if err := p.Build(ctx); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(ctx.DataDir, "filters.json")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	var idx FacetIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		t.Fatal(err)
	}
	if len(idx.Kitchenware) == 0 {
		t.Fatal("index should not be empty")
	}
}

func TestStatsCollectorBuild(t *testing.T) {
	ctx := newTestContext(t, testRecipes(), true)
	p := &StatsCollector{}
	if err := p.Build(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(ctx.DataDir, "meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.Total != 3 {
		t.Errorf("total = %d, want 3", m.Total)
	}
	if m.Tags["热菜"] != 2 {
		t.Errorf("tags[热菜] = %d, want 2", m.Tags["热菜"])
	}
	if m.Kitchenware["炒锅"] != 2 {
		t.Errorf("kitchenware[炒锅] = %d, want 2", m.Kitchenware["炒锅"])
	}

	// all.json 也应存在
	if _, err := os.Stat(filepath.Join(ctx.DataDir, "all.json")); err != nil {
		t.Fatal("all.json should exist")
	}
}

func TestTagIndexerBuild(t *testing.T) {
	ctx := newTestContext(t, testRecipes(), true)
	p := &TagIndexer{}
	if err := p.Build(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(ctx.DataDir, "by-tag", "热菜.json"))
	if err != nil {
		t.Fatal(err)
	}
	var list []RecipeSummary
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Errorf("热菜 group = %d, want 2", len(list))
	}
}

func TestDetailSplitterBuild(t *testing.T) {
	ctx := newTestContext(t, testRecipes(), true)
	p := &DetailSplitter{}
	if err := p.Build(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(ctx.DataDir, "details", "r1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var r models.Recipe
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	if r.Name != "红烧肉" {
		t.Errorf("unexpected detail: %+v", r)
	}
}

func TestAIExporterBuild(t *testing.T) {
	cfg := models.DefaultConfig()
	cfg.AISnapshot = true
	ctx := newTestContext(t, testRecipes(), true)
	ctx.Config = cfg
	p := &AIExporter{}
	if err := p.Build(ctx); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(ctx.DataDir, "ai-corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	lines := splitLines(string(data))
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	var entry AICorpusEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Name == "" || len(entry.MainIngredients) == 0 {
		t.Errorf("unexpected AI entry: %+v", entry)
	}
}

func TestCacheReuse(t *testing.T) {
	// 第一次构建生成缓存，第二次（未 force）应直接复用
	ctx := newTestContext(t, testRecipes(), true)
	p := &FacetIndexer{}
	if err := p.Build(ctx); err != nil {
		t.Fatal(err)
	}
	// 重新创建上下文（同一缓存目录），不 force
	ctx2 := pipeline.NewBuildContext(testRecipes(), models.DefaultConfig(), ctx.OutputDir, ctx.CacheRoot, ctx.ConfigPath, false)
	if err := p.Build(ctx2); err != nil {
		t.Fatal(err)
	}
	if !ctx2.Cache.IsValid("facet_indexer", computeDeps(ctx2, nil), ctx2.Config.PluginTTL("facet_indexer")) {
		t.Fatal("cache should be valid after reuse")
	}
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			out = append(out, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
