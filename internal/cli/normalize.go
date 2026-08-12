package cli

import "flavor-vault/internal/models"

// normalizeMultiVersion 将菜谱统一为多版本结构：
// 内容（食材/调料/步骤/media/统计）一律放 versions（至少 1 个版本），
// 顶层只保留元数据（id/name/description/tags/kitchenware/时间戳）。
// 历史单版本顶层承载结构（无 versions）会迁移到 versions[0]，顶层内容字段清空。
// 幂等：已是多版本（len(Versions)>0）则不动。
// fv add / fv edit 均调用，保证所有菜谱（含历史数据）最终都是多版本结构。
func normalizeMultiVersion(r *models.Recipe) {
	if len(r.Versions) > 0 {
		return
	}
	r.Versions = []models.Version{{
		Ingredients: r.Ingredients,
		Seasonings:  r.Seasonings,
		Steps:       r.Steps,
		Media:       r.Media,
		Stats:       r.Stats,
	}}
	r.Ingredients = models.Ingredients{}
	r.Seasonings = nil
	r.Steps = nil
	r.Media = models.Media{}
	r.Stats = models.Stats{}
}
