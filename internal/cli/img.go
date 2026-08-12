package cli

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"flavor-vault/internal/models"
)

// CJK 字体候选（跨平台系统字体）
var cjkFontCandidates = []string{
	// Linux
	"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
	"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
	"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
	"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
	// macOS
	"/System/Library/Fonts/PingFang.ttc",
	"/System/Library/Fonts/STHeiti Light.ttc",
	"/System/Library/Fonts/Hiragino Sans GB.ttc",
	// Windows
	`C:\Windows\Fonts\msyh.ttc`,
	`C:\Windows\Fonts\simhei.ttf`,
	`C:\Windows\Fonts\simsun.ttc`,
}

// 调色板
var (
	colTitle   = color.RGBA{0x1f, 0x29, 0x37, 0xff} // 标题深灰
	colBody    = color.RGBA{0x37, 0x41, 0x51, 0xff} // 正文
	colMuted   = color.RGBA{0x6b, 0x72, 0x80, 0xff} // 次要灰
	colAccent  = color.RGBA{0x05, 0x96, 0x69, 0xff} // 强调绿（食材/调料标题）
	colTag     = color.RGBA{0xd9, 0x77, 0x06, 0xff} // 标签橙
	colLine    = color.RGBA{0xe5, 0xe7, 0xeb, 0xff} // 分隔线
	colBG      = color.RGBA{0xff, 0xff, 0xff, 0xff} // 背景白
)

// loadCJKFont 跨平台加载中文字体（ttc/ttf 均可）
func loadCJKFont() (*opentype.Font, error) {
	for _, p := range cjkFontCandidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if f, err := opentype.Parse(data); err == nil {
			return f, nil
		}
		if coll, err := opentype.ParseCollection(data); err == nil && coll.NumFonts() > 0 {
			if f, err := coll.Font(0); err == nil {
				return f, nil
			}
		}
	}
	return nil, fmt.Errorf("未找到可用中文字体（已尝试 %d 个系统字体路径）；请安装任意 CJK 字体", len(cjkFontCandidates))
}

// painter 长图画布
type painter struct {
	img   *image.RGBA
	font  *opentype.Font
	faces map[float64]font.Face
	W     int  // 画布宽
	maxW  int  // 内容最大宽
}

func newPainter(f *opentype.Font, width, pad int) *painter {
	return &painter{
		img:   image.NewRGBA(image.Rect(0, 0, width, 9000)),
		font:  f,
		faces: map[float64]font.Face{},
		W:     width,
		maxW:  width - 2*pad,
	}
}

// face 按字号缓存字体
func (p *painter) face(size float64) font.Face {
	if f, ok := p.faces[size]; ok {
		return f
	}
	f, err := opentype.NewFace(p.font, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}
	p.faces[size] = f
	return f
}

func (p *painter) lineHeight(size float64) int {
	return p.face(size).Metrics().Height.Ceil()
}

// wrap 按最大宽度换行（中文按字符断行）
func (p *painter) wrap(text string, size float64, maxW int) []string {
	f := p.face(size)
	var lines []string
	cur := ""
	for _, r := range text {
		t := cur + string(r)
		if cur != "" && font.MeasureString(f, t).Ceil() > maxW {
			lines = append(lines, cur)
			cur = string(r)
		} else {
			cur = t
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// text 绘制文本（自动换行），返回下一个可用 y（下一行顶部）
func (p *painter) text(text string, x, y int, size float64, col color.RGBA, maxW int) int {
	f := p.face(size)
	ascent := f.Metrics().Ascent.Ceil()
	yy := y
	d := &font.Drawer{Dst: p.img, Src: image.NewUniform(col), Face: f}
	for _, ln := range p.wrap(text, size, maxW) {
		d.Dot = fixed.P(x, yy+ascent)
		d.DrawString(ln)
		yy += p.lineHeight(size)
	}
	return yy
}

// rect 填充矩形（用于分隔线/标签底）
func (p *painter) rect(x, y, w, h int, col color.RGBA) {
	draw.Draw(p.img, image.Rect(x, y, x+w, y+h), image.NewUniform(col), image.Point{}, draw.Src)
}

// renderShareImage 将菜谱渲染为竖版分享长图（PNG），写入 outPath。
// siteRoot 非空时在底部标注菜谱页面链接。
func renderShareImage(r *models.Recipe, siteRoot, outPath string) error {
	f, err := loadCJKFont()
	if err != nil {
		return err
	}
	const (
		W   = 840
		pad = 48
	)
	p := newPainter(f, W, pad)
	x := pad
	draw.Draw(p.img, p.img.Bounds(), image.NewUniform(colBG), image.Point{}, draw.Src)

	y := pad

	// 标题
	y = p.text(r.Name, x, y, 36, colTitle, p.maxW) + 6

	// 简介
	if strings.TrimSpace(r.Description) != "" {
		y = p.text(r.Description, x, y, 17, colMuted, p.maxW) + 10
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
		y = p.text(strings.Join(meta, "  "), x, y, 15, colTag, p.maxW) + 8
	}

	versions := r.VersionsEffective()
	for vi, v := range versions {
		if len(versions) > 1 {
			name := strings.TrimSpace(v.Name)
			if name == "" {
				name = fmt.Sprintf("版本 %d", vi+1)
			}
			y += 8
			y = p.text("◆ "+name, x, y, 20, colTitle, p.maxW) + 6
		}

		// 统计
		if v.Stats.PrepTime > 0 || v.Stats.CookTime > 0 {
			total := v.Stats.PrepTime + v.Stats.CookTime
			stat := fmt.Sprintf("⏱ 准备 %d 分钟 · 烹饪 %d 分钟 · 总耗时 %d 分钟", v.Stats.PrepTime, v.Stats.CookTime, total)
			if v.Stats.Difficulty > 0 && v.Stats.Difficulty <= 5 {
				stat += fmt.Sprintf(" · 难度 %s", strings.Repeat("★", v.Stats.Difficulty))
			}
			y = p.text(stat, x, y, 15, colMuted, p.maxW) + 8
		}

		// 主要食材
		if len(v.Ingredients.Main) > 0 {
			y = p.text("🥘 主要食材", x, y, 21, colAccent, p.maxW) + 2
			for _, ing := range v.Ingredients.Main {
				y = p.text("· "+shareIngredient(ing), x+16, y, 16, colBody, p.maxW-16)
			}
			y += 6
		}
		// 配菜 / 辅料
		if len(v.Ingredients.Side) > 0 {
			y = p.text("🥬 配菜 / 辅料", x, y, 21, colAccent, p.maxW) + 2
			for _, ing := range v.Ingredients.Side {
				y = p.text("· "+shareIngredient(ing), x+16, y, 16, colBody, p.maxW-16)
			}
			y += 6
		}
		// 调料
		if len(v.Seasonings) > 0 {
			y = p.text("🧂 调料", x, y, 21, colAccent, p.maxW) + 2
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
				y = p.text("· "+strings.Join(parts, " "), x+16, y, 16, colBody, p.maxW-16)
			}
			y += 6
		}
		// 步骤
		if len(v.Steps) > 0 {
			y += 4
			y = p.text("📋 步骤", x, y, 21, colAccent, p.maxW) + 2
			for _, s := range v.Steps {
				desc := strings.TrimSpace(s.Description)
				if desc == "" {
					continue
				}
				order := s.Order
				if order <= 0 {
					order = 0
				}
				prefix := "- "
				if order > 0 {
					prefix = fmt.Sprintf("%d. ", order)
				}
				y = p.text(prefix+desc, x+16, y, 16, colBody, p.maxW-16)
			}
			y += 6
		}
	}

	// 底部链接
	if siteRoot != "" {
		y += 4
		p.rect(x, y, p.maxW, 2, colLine)
		y += 14
		y = p.text("👉 完整菜谱："+siteRoot+"/recipe/"+r.ID, x, y, 13, colMuted, p.maxW)
	}

	end := y + pad
	if end > 9000 {
		end = 9000
	}
	out := p.img.SubImage(image.Rect(0, 0, W, end)).(*image.RGBA)

	fh, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer fh.Close()
	if err := png.Encode(fh, out); err != nil {
		return fmt.Errorf("编码 PNG 失败: %w", err)
	}
	return nil
}
