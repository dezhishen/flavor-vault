package cli

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"flavor-vault/internal/models"
)

// TestRenderShareImage 验证 share --img 生成的 PNG 有效（需系统有 CJK 字体，否则跳过）
func TestRenderShareImage(t *testing.T) {
	if _, err := loadCJKFont(); err != nil {
		t.Skipf("无可用 CJK 字体，跳过: %v", err)
	}
	r := &models.Recipe{
		Name:        "红烧肉",
		Description: "肥而不腻，入口即化",
		Tags:        []string{"热菜", "下饭"},
		Kitchenware: []string{"炒锅"},
		Versions: []models.Version{{
			Name: "经典版",
			Stats: models.Stats{PrepTime: 10, CookTime: 60, Difficulty: 3},
			Ingredients: models.Ingredients{
				Main: []models.Ingredient{{Name: "五花肉", Amount: "500g", Note: "带皮", Alternatives: []models.IngredientOption{{Name: "梅花肉", Amount: "500g", Note: "代替五花肉"}}}},
				Side: []models.Ingredient{{Name: "姜", Amount: "5片"}},
			},
			Seasonings: []models.Seasoning{{Name: "香葱", Amount: "2根", Alternatives: []models.SeasoningOption{{Name: "香菜", Note: "代替香葱"}}}},
			Steps:      []models.Step{{Order: 1, Description: "五花肉切块焯水"}, {Order: 2, Description: "小火煸出油，加生抽调味"}, {Order: 3, Description: "加水没过肉，小火慢炖一小时以上，收汁出锅"}},
		}},
	}
	out := filepath.Join(t.TempDir(), "share.png")
	if err := renderShareImage(r, "https://fv.sdniu.top", out); err != nil {
		t.Fatalf("renderShareImage 失败: %v", err)
	}
	f, err := os.Open(out)
	if err != nil {
		t.Fatalf("打开图片失败: %v", err)
	}
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatalf("生成的图片无效: %v", err)
	}
	if format != "png" || cfg.Width <= 0 || cfg.Height <= 0 {
		t.Fatalf("非 PNG 或尺寸异常: %s %dx%d", format, cfg.Width, cfg.Height)
	}
}
