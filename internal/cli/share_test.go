package cli

import (
	"strings"
	"testing"

	"flavor-vault/internal/models"
)

// TestShareText 验证分享文案渲染：标题/简介/标签/统计/食材（含可替换）/调料（含备选）/步骤
func TestShareText(t *testing.T) {
	r := &models.Recipe{
		Name:        "红烧肉",
		Description: "肥而不腻，入口即化",
		Tags:        []string{"热菜", "下饭"},
		Kitchenware: []string{"炒锅"},
		Versions: []models.Version{{
			Name: "经典版",
			Stats: models.Stats{
				PrepTime:   10,
				CookTime:   60,
				Difficulty: 3,
			},
			Ingredients: models.Ingredients{
				Main: []models.Ingredient{{
					Name: "五花肉", Amount: "500g", Note: "带皮",
					Alternatives: []models.IngredientOption{{Name: "梅花肉", Amount: "500g", Note: "代替五花肉"}},
				}},
				Side: []models.Ingredient{{Name: "姜", Amount: "5片"}},
			},
			Seasonings: []models.Seasoning{{
				Name: "香葱", Amount: "2根",
				Alternatives: []models.SeasoningOption{{Name: "香菜", Note: "代替香葱"}},
			}},
			Steps: []models.Step{{Order: 1, Description: "五花肉切块焯水"}, {Order: 2, Description: "小火煸出油"}},
		}},
	}

	out := shareText(r)
	for _, want := range []string{
		"# 🍳 红烧肉",
		"肥而不腻，入口即化",
		"#热菜",
		"🔧 炒锅",
		"⏱ 准备 10 分钟 · 烹饪 60 分钟 · 总耗时 70 分钟",
		"难度 ★★★",
		"## 🥘 主要食材",
		"可换 梅花肉 500g（代替五花肉）",
		"## 🥬 配菜 / 辅料",
		"## 🧂 调料",
		"可换 香菜（代替香葱）",
		"## 📋 步骤",
		"1. 五花肉切块焯水",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("分享文案缺少 %q：\n%s", want, out)
		}
	}
}

// TestShareTextMultiVersion 验证多版本时各版本都有标题
func TestShareTextMultiVersion(t *testing.T) {
	r := &models.Recipe{
		Name: "素菜高汤",
		Versions: []models.Version{
			{Name: "经典版", Ingredients: models.Ingredients{Main: []models.Ingredient{{Name: "西红柿"}}}, Steps: []models.Step{{Order: 1, Description: "炒出沙"}}},
			{Name: "免辣版", Ingredients: models.Ingredients{Main: []models.Ingredient{{Name: "冬瓜"}}}, Steps: []models.Step{{Order: 1, Description: "清水煮"}}},
		},
	}
	out := shareText(r)
	if !strings.Contains(out, "## 经典版") || !strings.Contains(out, "## 免辣版") {
		t.Fatalf("多版本缺少版本标题：\n%s", out)
	}
}

// TestShareTextFallback 验证无统计/无食材时不崩溃、不输出空标题
func TestShareTextFallback(t *testing.T) {
	r := &models.Recipe{Name: "空菜谱"}
	out := shareText(r)
	if !strings.Contains(out, "# 🍳 空菜谱") {
		t.Fatalf("空菜谱标题缺失：\n%s", out)
	}
	if strings.Contains(out, "## 🥘") {
		t.Fatalf("无食材不应输出主要食材标题：\n%s", out)
	}
}
