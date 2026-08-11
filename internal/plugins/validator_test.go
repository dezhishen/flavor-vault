package plugins

import (
	"strings"
	"testing"

	"flavor-vault/internal/models"
)

// TestValidateRecipeVersions 校验多版本：每个版本需主要食材/步骤/difficulty；调料与备选需名称
func TestValidateRecipeVersions(t *testing.T) {
	// 合法多版本
	ok := &models.Recipe{Name: "红烧肉", Versions: []models.Version{
		{Name: "经典版", Ingredients: models.Ingredients{Main: []models.Ingredient{{Name: "五花肉"}}},
			Seasonings: []models.Seasoning{{Name: "香葱", Alternatives: []models.SeasoningOption{{Name: "香菜"}}}},
			Steps:      []models.Step{{Order: 1, Description: "焯水"}}, Stats: models.Stats{Difficulty: 3}},
		{Name: "少油版", Ingredients: models.Ingredients{Main: []models.Ingredient{{Name: "白萝卜"}}},
			Steps: []models.Step{{Order: 1, Description: "煎"}}, Stats: models.Stats{Difficulty: 2}},
	}}
	if err := ValidateRecipe(ok, nil); err != nil {
		t.Fatalf("合法多版本应通过: %v", err)
	}

	// 某版本缺主要食材 → 报错
	bad := &models.Recipe{Name: "x", Versions: []models.Version{
		{Name: "v1", Ingredients: models.Ingredients{Main: nil}, Steps: []models.Step{{Order: 1, Description: "s"}}, Stats: models.Stats{Difficulty: 1}},
	}}
	err := ValidateRecipe(bad, nil)
	if err == nil || !strings.Contains(err.Error(), "v1") || !strings.Contains(err.Error(), "ingredients.main") {
		t.Fatalf("应报错并指出版本 v1 缺主料: %v", err)
	}

	// 调料备选缺 name → 报错
	bad2 := &models.Recipe{Name: "x", Ingredients: models.Ingredients{Main: []models.Ingredient{{Name: "a"}}},
		Seasonings: []models.Seasoning{{Name: "香葱", Alternatives: []models.SeasoningOption{{Name: ""}}}},
		Steps:      []models.Step{{Order: 1, Description: "s"}}, Stats: models.Stats{Difficulty: 1}}
	if err := ValidateRecipe(bad2, nil); err == nil {
		t.Fatal("调料备选缺 name 应报错")
	}

	// 单版本（无 versions）仍按顶层校验
	old := &models.Recipe{Name: "旧", Ingredients: models.Ingredients{Main: []models.Ingredient{{Name: "a"}}},
		Steps: []models.Step{{Order: 1, Description: "s"}}, Stats: models.Stats{Difficulty: 2}}
	if err := ValidateRecipe(old, nil); err != nil {
		t.Fatalf("单版本应通过: %v", err)
	}
}
