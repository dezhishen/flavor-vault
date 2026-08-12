package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"flavor-vault/internal/data"
	"flavor-vault/internal/models"
	"flavor-vault/internal/store"
)

// newShareCmd 生成本地可发送的菜谱分享消息（Markdown 文本），
// 可直接复制发给 IM / AI 助手；--out 可写入文件。
func newShareCmd() *cobra.Command {
	var (
		outFile string
		imgPath string
		noImg   bool
	)
	cmd := &cobra.Command{
		Use:   "share <id>",
		Short: "生成本地可发送的菜谱分享消息（Markdown 带图 / PNG 长图，可直接发给 IM / AI 助手）",
		Long: `生成菜谱的分享内容：
- 默认输出 Markdown（嵌入封面图与步骤图 + 完整菜谱链接），适合支持 Markdown 的 IM（钉钉/飞书/语雀等）；
- --img 可导出 PNG 分享长图（竖版，含标题/简介/统计/食材/调料/步骤），适合不支持带图 Markdown 的平台；
- --no-img 输出纯文字 Markdown；数据来源：优先本地 recipes/<id>.json，否则读取部署的 details/<id>.json。`,
		Example: `  fv share chao-jue-zi-su-xia                    # 打印带图 Markdown
  fv share chao-jue-zi-su-xia --out ~/share.md     # 写入 md 文件
  fv share chao-jue-zi-su-xia --img ~/share.png    # 导出 PNG 分享长图
  fv share chao-jue-zi-su-xia --no-img             # 纯文字不带图`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}

			// 1. 优先本地菜谱文件（维护者模式，数据最新）
			var r *models.Recipe
			remote := false
			locator := ""
			fs := store.NewRecipeFileStore(recipesDir(cfg, projectRoot))
			if rr, err := fs.Load(id); err == nil {
				r = rr
			} else {
				// 2. 回退读取部署的 details/<id>.json（使用者模式）
				locator, remote = data.Locator(cfg, projectRoot, "details/"+id+".json")
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

			assetBase := ""
			siteRoot := ""
			if !noImg {
				assetBase = shareAssetBase(remote, locator, cfg, projectRoot, id)
			}
			siteRoot = shareSiteRoot(remote, locator, cfg, projectRoot, id)
			text := shareText(r, assetBase)
			if siteRoot != "" {
				text += fmt.Sprintf("\n---\n👉 完整菜谱：%s/recipe/%s\n", siteRoot, id)
			}
			if strings.TrimSpace(imgPath) != "" {
				if err := renderShareImage(r, siteRoot, imgPath, shareImageLoader(remote, locator, cfg, projectRoot, id)); err != nil {
					return fmt.Errorf("生成分享图片失败: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✔ 已生成分享图片到 %s\n", imgPath)
			}
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
	cmd.Flags().StringVar(&imgPath, "img", "", "导出菜谱分享长图（PNG），如 /tmp/share.png")
	cmd.Flags().BoolVar(&noImg, "no-img", false, "不嵌入图片（纯文字 Markdown）")
	return cmd
}

// shareEndpoints 解析数据 endpoint（远程详情反推 / 本地 meta 注入 / 内置默认），返回去尾斜杠的地址
func shareEndpoints(remote bool, locator string, cfg *models.Config, projectRoot, id string) string {
	var endpoint string
	if remote && locator != "" {
		endpoint = strings.TrimSuffix(locator, "/details/"+id+".json")
	} else {
		endpoint = data.DefaultEndpointFromMeta(cfg, projectRoot)
		if endpoint == "" {
			endpoint = data.DefaultEndpoint
		}
	}
	return strings.TrimRight(strings.TrimSpace(endpoint), "/")
}

// shareSiteRoot 派生站点根地址（endpoint 去 /data，如 https://fv.sdniu.top），用于菜谱页面链接
func shareSiteRoot(remote bool, locator string, cfg *models.Config, projectRoot, id string) string {
	endpoint := shareEndpoints(remote, locator, cfg, projectRoot, id)
	if endpoint == "" {
		return ""
	}
	return strings.TrimSuffix(endpoint, "/data")
}

// shareAssetBase 派生菜谱图片资源的 Markdown URL 基址（如 https://fv.sdniu.top/assets）。
// endpoint 形如 https://fv.sdniu.top/data → 站点根 + /assets
func shareAssetBase(remote bool, locator string, cfg *models.Config, projectRoot, id string) string {
	siteRoot := shareSiteRoot(remote, locator, cfg, projectRoot, id)
	if siteRoot == "" {
		return ""
	}
	return siteRoot + "/assets"
}

// shareText 将菜谱渲染为 Markdown 分享文案；assetBase 非空时嵌入封面图与步骤图
func shareText(r *models.Recipe, assetBase string) string {
	var b strings.Builder
	base := strings.TrimRight(assetBase, "/")

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

		// 封面图（统计后、食材前）
		if base != "" && strings.TrimSpace(v.Media.Cover) != "" {
			fmt.Fprintf(&b, "\n![封面](%s/%s)\n", base, v.Media.Cover)
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
				// 步骤图
				if base != "" && strings.TrimSpace(s.ImageRef) != "" {
					fmt.Fprintf(&b, "   ![第 %d 步](%s/%s)\n", order, base, s.ImageRef)
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

// shareImageLoader 构造按 image_ref 加载图片的加载器，供分享长图嵌入步骤图。
// 优先本地文件（assets 目录 / cwd 相对），否则从 assetBase（线上站点资源）或直接 URL 下载。
func shareImageLoader(remote bool, locator string, cfg *models.Config, projectRoot, id string) imageLoader {
	assetBase := shareAssetBase(remote, locator, cfg, projectRoot, id)
	return func(ref string) (image.Image, error) {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			return nil, errors.New("空图片引用")
		}
		// 1) 本地文件：assets 目录（编辑暂存 / 本地构建 dist）/ cwd 相对
		bases := []string{assetDirFor(cfg, projectRoot), ".flavor-vault/assets", "dist/assets", "."}
		for _, base := range bases {
			p := filepath.ToSlash(filepath.Join(base, ref))
			if data, err := os.ReadFile(p); err == nil {
				if img, _, err := image.Decode(bytes.NewReader(data)); err == nil {
					return img, nil
				}
			}
		}
		// 2) 远程：ref 本身是 URL，或按 assetBase 拼资源地址
		if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
			return fetchImage(ref)
		}
		if assetBase != "" && !strings.HasPrefix(ref, "/") {
			return fetchImage(strings.TrimRight(assetBase, "/") + "/" + ref)
		}
		return nil, fmt.Errorf("无法加载图片 %s", ref)
	}
}

// fetchImage 从 URL 下载并解码图片
func fetchImage(url string) (image.Image, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	img, _, err := image.Decode(resp.Body)
	return img, err
}
