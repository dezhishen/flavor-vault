package plugins

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
	"flavor-vault/internal/pipeline"
)

// Validator 校验菜谱必填字段、标签合法性
type Validator struct {
	Logger func(format string, args ...interface{})
}

// Name 插件标识
func (v *Validator) Name() string { return "validator" }

// RegisterCommands validator 不注册子命令
func (v *Validator) RegisterCommands(_ *cobra.Command) error { return nil }

// Build 校验所有菜谱，不合法则中断构建
func (v *Validator) Build(ctx *pipeline.BuildContext) error {
	var errs []string

	for _, r := range ctx.Recipes {
		if err := validateRecipe(r); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", r.ID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("存在 %d 个校验错误:\n  - %s", len(errs), strings.Join(errs, "\n  - "))
	}
	if v.Logger != nil {
		v.Logger("✔ 校验通过，共 %d 道菜谱\n", len(ctx.Recipes))
	}
	return nil
}

// ValidateRecipe 校验单个菜谱（供 CLI 的 add/edit 命令复用）
func ValidateRecipe(r *models.Recipe, _ *models.Config) error {
	return validateRecipe(r)
}

func validateRecipe(r *models.Recipe) error {
	var problems []string
	if strings.TrimSpace(r.Name) == "" {
		problems = append(problems, "缺少 name（菜名）")
	}
	versions := r.VersionsEffective()
	if len(versions) == 0 {
		problems = append(problems, "缺少菜谱内容（versions 或默认版本字段）")
	}
	for i, v := range versions {
		label := fmt.Sprintf("版本[%d] ", i+1)
		if v.Name != "" {
			label = fmt.Sprintf("版本[%s] ", v.Name)
		}
		for _, p := range validateVersion(v) {
			problems = append(problems, label+p)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("校验失败: %s", strings.Join(problems, "; "))
}

// validateVersion 校验单个版本：主要食材/步骤/difficulty 必填；调料与备选方案需有名称。
func validateVersion(v models.Version) []string {
	var p []string
	if len(v.Ingredients.Main) == 0 {
		p = append(p, "缺少 ingredients.main（主要食材）")
	}
	if len(v.Steps) == 0 {
		p = append(p, "缺少 steps（步骤）")
	}
	if v.Stats.Difficulty < 1 || v.Stats.Difficulty > 5 {
		p = append(p, "difficulty 必须在 1-5 之间")
	}
	for _, s := range v.Seasonings {
		if strings.TrimSpace(s.Name) == "" {
			p = append(p, "调料缺少 name")
		}
		for _, alt := range s.Alternatives {
			if strings.TrimSpace(alt.Name) == "" {
				p = append(p, "调料备选方案缺少 name")
			}
		}
	}
	// 食材可替换方案需有名称
	ingredients := append(append([]models.Ingredient{}, v.Ingredients.Main...), v.Ingredients.Side...)
	for _, ing := range ingredients {
		for _, alt := range ing.Alternatives {
			if strings.TrimSpace(alt.Name) == "" {
				p = append(p, "食材备选方案缺少 name")
			}
		}
	}
	return p
}
