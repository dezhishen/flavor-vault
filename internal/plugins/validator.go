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
	whitelist := make(map[string]bool)
	for _, t := range ctx.Config.Tags {
		whitelist[t] = true
	}
	strictTags := len(ctx.Config.Tags) > 0

	for _, r := range ctx.Recipes {
		if err := validateRecipe(r, whitelist, strictTags); err != nil {
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
func ValidateRecipe(r *models.Recipe, cfg *models.Config) error {
	whitelist := make(map[string]bool)
	for _, t := range cfg.Tags {
		whitelist[t] = true
	}
	return validateRecipe(r, whitelist, len(cfg.Tags) > 0)
}

func validateRecipe(r *models.Recipe, whitelist map[string]bool, strictTags bool) error {
	var problems []string
	if strings.TrimSpace(r.Name) == "" {
		problems = append(problems, "缺少 name（菜名）")
	}
	if len(r.Ingredients.Main) == 0 {
		problems = append(problems, "缺少 ingredients.main（主要食材）")
	}
	if len(r.Steps) == 0 {
		problems = append(problems, "缺少 steps（步骤）")
	}
	if r.Stats.Difficulty < 1 || r.Stats.Difficulty > 5 {
		problems = append(problems, "difficulty 必须在 1-5 之间")
	}
	if strictTags {
		for _, t := range r.Tags {
			if !whitelist[t] {
				problems = append(problems, fmt.Sprintf("标签 %q 不在白名单中", t))
			}
		}
	}
	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("校验失败: %s", strings.Join(problems, "; "))
}
