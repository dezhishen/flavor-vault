package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/data"
	"flavor-vault/internal/models"
	"flavor-vault/internal/store"
)

// newShareCmd 生成本地可发送的菜谱分享消息（Markdown 文本），
// 可直接复制发给 IM / AI 助手；--out 可写入文件。
func newShareCmd() *cobra.Command {
	var outFile string
	cmd := &cobra.Command{
		Use:   "share <id>",
		Short: "生成本地可发送的菜谱分享消息（Markdown，可直接发给 IM / AI 助手）",
		Long: `生成菜谱的 Markdown 分享文案（标题/简介/标签/统计/食材/调料/步骤），
可直接复制到微信、钉钉、飞书等 IM，或交给 AI 助手整理发送。
数据来源：优先本地 recipes/<id>.json（维护者模式），否则读取部署的 details/<id>.json（使用者模式）。`,
		Example: `  fv share chao-jue-zi-su-xia            # 打印分享消息
  fv share chao-jue-zi-su-xia --out ~/share.md  # 写入文件`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}

			// 1. 优先本地菜谱文件（维护者模式，数据最新）
			var r *models.Recipe
			fs := store.NewRecipeFileStore(recipesDir(cfg, projectRoot))
			if rr, err := fs.Load(id); err == nil {
				r = rr
			} else {
				// 2. 回退读取部署的 details/<id>.json（使用者模式）
				locator, remote := data.Locator(cfg, projectRoot, "details/"+id+".json")
				raw, err := data.ReadJSON(locator, remote)
				if err != nil {
					return err
				}
				var rr models.Recipe
				if err := json.Unmarshal(raw, &rr); err != nil {
					return fmt.Errorf("菜谱 %s 解析失败: %w", id, err)
				}
				r = &rr
			}

			text := shareText(r)
			if strings.TrimSpace(outFile) != "" {
				if err := os.WriteFile(outFile, []byte(text), 0o644); err != nil {
					return fmt.Errorf("写入文件失败: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✔ 已生成分享消息到 %s（%d 字符）\n", outFile, len(text))
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), text)
			return nil
		},
	}
	cmd.Flags().StringVar(&outFile, "out", "", "写入文件路径（默认打印到终端）")
	return cmd
}

// shareText 将菜谱渲染为 Markdown 分享文案
func shareText(r *models.Recipe) string {
	var b strings.Builder

	// 标题 + 简介
	fmt.Fprintf(&b, "# 🍳 %s\n", r.Name)
	if strings.TrimSpace(r.Description) != "" {
		fmt.Fprintf(&b, "\n> %s\n", r.Description)
	}

	// 标签 / 厨具
	var meta []string
	for _, t := range r.Tags {
		if strings.TrimSpace(t) != "" {
			meta = append(meta, "#"+strings.TrimSpace(t))
		}
	}
	for _, k := range r.Kitchenware {
		if strings.TrimSpace(k) != "" {
			meta = append(meta, "🔧 "+strings.TrimSpace(k))
		}
	}
	if len(meta) > 0 {
		fmt.Fprintf(&b, "\n%s\n", strings.Join(meta, "  "))
	}

	versions := r.VersionsEffective()
	for vi, v := range versions {
		if len(versions) > 1 {
			name := strings.TrimSpace(v.Name)
			if name == "" {
				name = fmt.Sprintf("版本 %d", vi+1)
			}
			fmt.Fprintf(&b, "\n## %s\n", name)
		}

		// 统计
		if v.Stats.PrepTime > 0 || v.Stats.CookTime > 0 {
			total := v.Stats.PrepTime + v.Stats.CookTime
			fmt.Fprintf(&b, "\n> ⏱ 准备 %d 分钟 · 烹饪 %d 分钟 · 总耗时 %d 分钟", v.Stats.PrepTime, v.Stats.CookTime, total)
			if v.Stats.Difficulty > 0 && v.Stats.Difficulty <= 5 {
				fmt.Fprintf(&b, " · 难度 %s", strings.Repeat("★", v.Stats.Difficulty))
			}
			b.WriteString("\n")
		}

		// 主要食材
		if len(v.Ingredients.Main) > 0 {
			b.WriteString("\n## 🥘 主要食材\n")
			for _, ing := range v.Ingredients.Main {
				b.WriteString("- " + shareIngredient(ing) + "\n")
			}
		}
		// 配菜 / 辅料
		if len(v.Ingredients.Side) > 0 {
			b.WriteString("\n## 🥬 配菜 / 辅料\n")
			for _, ing := range v.Ingredients.Side {
				b.WriteString("- " + shareIngredient(ing) + "\n")
			}
		}
		// 调料
		if len(v.Seasonings) > 0 {
			b.WriteString("\n## 🧂 调料\n")
			for _, s := range v.Seasonings {
				parts := []string{s.Name}
				if s.Amount != "" {
					parts = append(parts, s.Amount)
				}
				if s.Note != "" {
					parts = append(parts, "（"+s.Note+"）")
				}
				if len(s.Alternatives) > 0 {
					parts = append(parts, "可换 "+shareSeasoningOptions(s.Alternatives))
				}
				b.WriteString("- " + strings.Join(parts, " ") + "\n")
			}
		}
		// 步骤
		if len(v.Steps) > 0 {
			b.WriteString("\n## 📋 步骤\n")
			for _, s := range v.Steps {
				desc := strings.TrimSpace(s.Description)
				if desc == "" {
					continue
				}
				order := s.Order
				if order <= 0 {
					order = 0 // 兜底：无序号时直接列点
				}
				if order > 0 {
					fmt.Fprintf(&b, "%d. %s\n", order, desc)
				} else {
					fmt.Fprintf(&b, "- %s\n", desc)
				}
			}
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

// shareIngredient 格式化单个食材条目（名称/用量/备注/可替换）
func shareIngredient(ing models.Ingredient) string {
	parts := []string{ing.Name}
	if ing.Amount != "" {
		parts = append(parts, ing.Amount)
	}
	if ing.Note != "" {
		parts = append(parts, "（"+ing.Note+"）")
	}
	if len(ing.Alternatives) > 0 {
		parts = append(parts, "可换 "+shareIngredientOptions(ing.Alternatives))
	}
	return strings.Join(parts, " ")
}

// shareIngredientOptions 格式化食材可替换方案为 "甲 用量/乙 用量" 形式
func shareIngredientOptions(opts []models.IngredientOption) string {
	var out []string
	for _, o := range opts {
		s := o.Name
		if s == "" {
			continue
		}
		if o.Amount != "" {
			s += " " + o.Amount
		}
		if o.Note != "" {
			s += "（" + o.Note + "）"
		}
		out = append(out, s)
	}
	return strings.Join(out, "/")
}

// shareSeasoningOptions 格式化调料备选方案为 "甲 用量/乙 用量" 形式
func shareSeasoningOptions(opts []models.SeasoningOption) string {
	var out []string
	for _, o := range opts {
		s := o.Name
		if s == "" {
			continue
		}
		if o.Amount != "" {
			s += " " + o.Amount
		}
		if o.Note != "" {
			s += "（" + o.Note + "）"
		}
		out = append(out, s)
	}
	return strings.Join(out, "/")
}
