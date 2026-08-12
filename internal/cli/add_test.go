package cli

import (
	"bufio"
	"strings"
	"testing"

	"flavor-vault/internal/models"
)

// TestPromptAddRecipeMultiVersion 验证 fv add 交互式多版本：默认版本 + 追加少油版 → versions 结构
func TestPromptAddRecipeMultiVersion(t *testing.T) {
	// 输入序列：菜名/ID(回车自动生成)/简介/标签/厨具 → 版本1 主料(材料) → 无配菜/可选/调料 → 步骤1(做)+配图空 →
	// 统计默认 → 添加其他版本(y) → 少油版 主料(材料2) → 步骤1(少油做) → 统计默认 → 继续版本(n)
	input := "测试菜\n\n\n\n\ny\n材料\n1份\nn\nn\nn\nn\n做\n\n\n\n\n\ny\n少油版\ny\n材料2\n1份\nn\nn\nn\nn\n少油做\n\n\n\n\n\nn\n\n"
	reader := bufio.NewReader(strings.NewReader(input))
	r, err := promptAddRecipe(reader, &models.Config{}, t.TempDir(), nil)
	if err != nil {
		t.Fatalf("promptAddRecipe err: %v", err)
	}
	if len(r.Versions) != 2 {
		t.Fatalf("期望 2 个版本，得到 %d", len(r.Versions))
	}
	if r.Versions[0].Name != "" || len(r.Versions[0].Ingredients.Main) != 1 || r.Versions[0].Ingredients.Main[0].Name != "材料" {
		t.Fatalf("默认版本内容错误: %+v", r.Versions[0])
	}
	if r.Versions[1].Name != "少油版" || len(r.Versions[1].Ingredients.Main) != 1 || r.Versions[1].Ingredients.Main[0].Name != "材料2" {
		t.Fatalf("少油版内容错误: %+v", r.Versions[1])
	}
	if len(r.Steps) != 0 || len(r.Ingredients.Main) != 0 {
		t.Fatalf("多版本时顶层内容应清空: %+v", r)
	}
}

// TestPromptAddRecipeSingleVersion 验证单版本（不添加其他版本）→ 保留顶层结构
func TestPromptAddRecipeSingleVersion(t *testing.T) {
	// 菜名/ID(回车自动生成)/简介/标签/厨具 → 版本1 主料(材料) → 无配菜/可选/调料 → 步骤1(做)+配图空 → 统计默认 → 不添加其他版本(n)
	input := "测试菜\n\n\n\n\ny\n材料\n1份\nn\nn\nn\nn\n做\n\n\n\n\n\nn\n\n"
	reader := bufio.NewReader(strings.NewReader(input))
	r, err := promptAddRecipe(reader, &models.Config{}, t.TempDir(), nil)
	if err != nil {
		t.Fatalf("promptAddRecipe err: %v", err)
	}
	if len(r.Versions) != 0 {
		t.Fatalf("单版本不应有 versions，得到 %d", len(r.Versions))
	}
	if len(r.Ingredients.Main) != 1 || r.Ingredients.Main[0].Name != "材料" {
		t.Fatalf("顶层主要食材错误: %+v", r.Ingredients)
	}
}
