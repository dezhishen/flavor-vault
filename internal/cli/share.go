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
		format  string
	)
	cmd := &cobra.Command{
		Use:   "share <id>",
		Short: "生成菜谱分享内容（Markdown / PNG 长图，可直接发 IM / AI 助手）",
		Long: `生成菜谱分享内容，导出格式与是否带图可控：

  --format md       Markdown（默认；--out 写文件，否则打印到终端）
  --format png      PNG 分享长图（竖版，含步骤图，底部带菜谱二维码）
  --format plain    纯文本（无 Markdown 标记，适合只支持纯文本的 IM/短信）
  --format all      Markdown + PNG 都导出

是否带图（Markdown）：默认嵌入封面图/步骤图 + 完整菜谱链接；--no-img 输出纯文字。

说明：
- 数据来源：优先本地 recipes/<id>.json（维护者模式），否则读取部署的 details/<id>.json
- PNG 输出路径：--img <file> 指定；未指定时用 --out 换 .png，再否则 <id>.png
- 图片资源 URL 从当前数据源 endpoint 派生；步骤图按 image_ref 从本地 assets 或线上加载`,
		Example: `  fv share chao-jue-zi-su-xia                                  # Markdown 带图，打印终端
  fv share chao-jue-zi-su-xia --format png                         # 导出 PNG 长图（chao-jue-zi-su-xia.png）
  fv share chao-jue-zi-su-xia --format png --img ~/share.png       # 指定 PNG 路径
  fv share chao-jue-zi-su-xia --format plain --out ~/share.txt     # 纯文本（无 Markdown 标记）
  fv share chao-jue-zi-su-xia --format all --out ~/a.md --img ~/b.png  # 同时导出 md + png
  fv share chao-jue-zi-su-xia --no-img                             # Markdown 纯文字（不带图）`,
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

			// 3. 校验并解析导出格式
			format = strings.ToLower(strings.TrimSpace(format))
			switch format {
			case "md", "png", "plain", "all":
			default:
				return fmt.Errorf("无效导出格式 %q（可选：md / png / plain / all）", format)
			}
			wantPlain := format == "plain"
			wantMD := format == "md" || format == "all"
			wantPNG := format == "png" || format == "all"
			if strings.TrimSpace(imgPath) != "" {
				wantPNG = true // 显式 --img 总是导出长图
			}

			// 4. 是否带图（Markdown 嵌入封面/步骤图）
			assetBase := ""
			if !noImg {
				assetBase = shareAssetBase(remote, locator, cfg, projectRoot, id)
			}
			siteRoot := shareSiteRoot(remote, locator, cfg, projectRoot, id)

			// 5. 导出纯文本（无 Markdown 标记，适合只支持纯文本的 IM）
			if wantPlain {
				text := shareTextPlain(r)
				if siteRoot != "" {
					text += "\n\n完整菜谱：" + siteRoot + "/recipe/" + id
				}
				if strings.TrimSpace(outFile) != "" {
					if err := ensureParentDir(outFile); err != nil {
						return fmt.Errorf("创建输出目录失败: %w", err)
					}
					if err := os.WriteFile(outFile, []byte(text), 0o644); err != nil {
						return fmt.Errorf("写入文件失败: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✔ 已生成分享文本到 %s（%d 字符）\n", outFile, len(text))
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), text)
				}
				return nil
			}

			// 6. 导出 Markdown
			if wantMD {
				text := shareText(r, assetBase)
				if siteRoot != "" {
					text += fmt.Sprintf("\n---\n👉 完整菜谱：%s/recipe/%s\n", siteRoot, id)
				}
				if strings.TrimSpace(outFile) != "" {
					if err := ensureParentDir(outFile); err != nil {
						return fmt.Errorf("创建输出目录失败: %w", err)
					}
					if err := os.WriteFile(outFile, []byte(text), 0o644); err != nil {
						return fmt.Errorf("写入文件失败: %w", err)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "✔ 已生成分享消息到 %s（%d 字符）\n", outFile, len(text))
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), text)
				}
			}

			// 6. 导出 PNG 长图
			if wantPNG {
				pngPath := strings.TrimSpace(imgPath)
				if pngPath == "" {
					if strings.TrimSpace(outFile) != "" {
						pngPath = swapExt(outFile, ".png")
					} else {
						pngPath = id + ".png"
					}
				}
				if err := ensureParentDir(pngPath); err != nil {
					return fmt.Errorf("创建图片输出目录失败: %w", err)
				}
				if err := renderShareImage(r, siteRoot, pngPath, shareImageLoader(remote, locator, cfg, projectRoot, id)); err != nil {
					return fmt.Errorf("生成分享图片失败: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✔ 已生成分享图片到 %s\n", pngPath)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&outFile, "out", "", "文本输出文件路径（默认打印到终端；父目录不存在会自动创建）")
	cmd.Flags().StringVar(&imgPath, "img", "", "PNG 分享长图输出路径（未指定时默认 <id>.png 或 --out 换 .png）")
	cmd.Flags().BoolVar(&noImg, "no-img", false, "Markdown 不嵌入图片（纯文字）")
	cmd.Flags().StringVar(&format, "format", "md", "导出格式：md（Markdown）/ png（PNG 长图）/ plain（纯文本）/ all（md+png）")
	return cmd
}

// ensureParentDir 确保文件所在目录存在（不存在则创建）
func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// swapExt 把 path 的扩展名替换为 ext（无扩展名则直接追加）
func swapExt(path, ext string) string {
	e := filepath.Ext(path)
	if e == "" {
		return path + ext
	}
	return strings.TrimSuffix(path, e) + ext
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

// shareTextPlain 将菜谱渲染为纯文本（无 Markdown 标记，适合只支持纯文本的 IM/短信）
func shareTextPlain(r *models.Recipe) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", r.Name)
	if strings.TrimSpace(r.Description) != "" {
		fmt.Fprintf(&b, "\n%s\n", r.Description)
	}
	var meta []string
	for _, t := range r.Tags {
		if strings.TrimSpace(t) != "" {
			meta = append(meta, strings.TrimSpace(t))
		}
	}
	for _, k := range r.Kitchenware {
		if strings.TrimSpace(k) != "" {
			meta = append(meta, "厨具 "+strings.TrimSpace(k))
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
			fmt.Fprintf(&b, "\n◆ %s\n", name)
		}
		if v.Stats.PrepTime > 0 || v.Stats.CookTime > 0 {
			total := v.Stats.PrepTime + v.Stats.CookTime
			fmt.Fprintf(&b, "\n准备 %d 分钟 · 烹饪 %d 分钟 · 总耗时 %d 分钟", v.Stats.PrepTime, v.Stats.CookTime, total)
			if v.Stats.Difficulty > 0 && v.Stats.Difficulty <= 5 {
				fmt.Fprintf(&b, " · 难度 %s", strings.Repeat("★", v.Stats.Difficulty))
			}
			b.WriteString("\n")
		}
		if len(v.Ingredients.Main) > 0 {
			b.WriteString("\n【主要食材】\n")
			for _, ing := range v.Ingredients.Main {
				b.WriteString(shareIngredient(ing) + "\n")
			}
		}
		if len(v.Ingredients.Side) > 0 {
			b.WriteString("\n【配菜 / 辅料】\n")
			for _, ing := range v.Ingredients.Side {
				b.WriteString(shareIngredient(ing) + "\n")
			}
		}
		if len(v.Seasonings) > 0 {
			b.WriteString("\n【调料】\n")
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
				b.WriteString(strings.Join(parts, " ") + "\n")
			}
		}
		if len(v.Steps) > 0 {
			b.WriteString("\n【步骤】\n")
			for _, s := range v.Steps {
				desc := strings.TrimSpace(s.Description)
				if desc == "" {
					continue
				}
				order := s.Order
				if order <= 0 {
					order = 0
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
