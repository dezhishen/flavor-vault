package cli

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/font/opentype"

	"flavor-vault/internal/models"
)

// TestRenderShareImage 验证 share --img 生成的 PNG 有效（需系统有 CJK 字体，否则跳过）
func TestRenderShareImage(t *testing.T) {
	if _, err := loadFonts(); err != nil {
		t.Skipf("无可用字体，跳过: %v", err)
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

// TestPainterGlyphCoverage 验证多字体回退链能覆盖分享图需要的全部字符（中文 + ASCII + 符号），
// 任何字符在链中都有字体可渲染，就不会出现方块字（.notdef）。
func TestPainterGlyphCoverage(t *testing.T) {
	fonts, err := loadFonts()
	if err != nil {
		t.Skipf("无可用字体，跳过: %v", err)
	}
	p := newPainter(fonts, 840, 48)
	need := "超绝紫苏虾分钟准备烹饪总耗时难度主要食材配菜辅料调料步骤完整菜谱可换代替省略" +
		"0123456789ABCXYZ" + "#./:·-()（）&%.★"
	for _, r := range need {
		ok := false
		for _, f := range fonts {
			if p.hasGlyph(f, r) {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("字符 %q（U+%04X）在回退链中无字体可渲染，将显示为方块", r, r)
		}
	}
}

// TestEmbeddedFontsFallback 验证内置字体（go:embed）有效，且仅用内置字体也能覆盖
// 分享图需要的全部字符——即使系统没有任何中文字体（精简 Linux/服务器/Docker），
// fv 二进制自带字体仍可正常渲染，不会出方块。
func TestEmbeddedFontsFallback(t *testing.T) {
	cjk := parseEmbedded(embeddedCJK)
	latin := parseEmbedded(embeddedLatin)
	if cjk == nil || latin == nil {
		t.Fatalf("内置字体解析失败：CJK=%v Latin=%v", cjk, latin)
	}
	p := newPainter([]*opentype.Font{cjk, latin}, 840, 48)
	need := "超绝紫苏虾分钟准备烹饪总耗时难度主要食材配菜辅料调料步骤完整菜谱可换代替省略" +
		"0123456789ABCXYZ" + "#./:·-()（）&%.★"
	for _, r := range need {
		ok := false
		for _, f := range []*opentype.Font{cjk, latin} {
			if p.hasGlyph(f, r) {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("内置字体链缺字符 %q（U+%04X），无系统字体时将为方块", r, r)
		}
	}
}
