package plugins

import "flavor-vault/internal/models"

// RecipeSummary 轻量菜谱信息（用于列表展示，不包含步骤）
type RecipeSummary struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Kitchenware []string `json:"kitchenware"`
	Ingredients []string `json:"ingredients"` // 主要食材名
	Cover       string   `json:"cover"`
	PrepTime    int      `json:"prep_time"`
	CookTime    int      `json:"cook_time"`
	Difficulty  int      `json:"difficulty"`
}

// ToSummary 从完整菜谱生成轻量摘要（内容取有效版本：versions[0] 或顶层默认版本）
func ToSummary(r *models.Recipe) RecipeSummary {
	v := r.VersionsEffective()[0]
	return RecipeSummary{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Tags:        r.Tags,
		Kitchenware: r.Kitchenware,
		Ingredients: r.MainIngredientNames(),
		Cover:       v.Media.Cover,
		PrepTime:    v.Stats.PrepTime,
		CookTime:    v.Stats.CookTime,
		Difficulty:  v.Stats.Difficulty,
	}
}

// Summaries 批量转换
func Summaries(recipes []*models.Recipe) []RecipeSummary {
	out := make([]RecipeSummary, 0, len(recipes))
	for _, r := range recipes {
		out = append(out, ToSummary(r))
	}
	return out
}
