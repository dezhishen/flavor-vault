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
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"

	"flavor-vault/internal/models"
)

// CJK 字体候选（跨平台系统字体；注意 wqy-zenhei.ttc 在新版 x/image 解析失败，故排在 Droid 之后）
var cjkFontCandidates = []string{
	// Linux
	"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
	"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
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

// Latin/ASCII 字体候选（覆盖数字/字母/标点，CJK 字体大多缺 ASCII）
var latinFontCandidates = []string{
	// Linux
	"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
	"/usr/share/fonts/truetype/freefont/FreeSans.ttf",
	// macOS
	"/System/Library/Fonts/Helvetica.ttc",
	"/System/Library/Fonts/HelveticaNeue.ttc",
	// Windows
	`C:\Windows\Fonts\arial.ttf`,
	`C:\Windows\Fonts\segoeui.ttf`,
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

// loadFont 从候选路径解析第一个可用字体（ttc/ttf 均可）
func loadFont(candidates []string) *opentype.Font {
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if f, err := opentype.Parse(data); err == nil {
			return f
		}
		if coll, err := opentype.ParseCollection(data); err == nil && coll.NumFonts() > 0 {
			if f, err := coll.Font(0); err == nil {
				return f
			}
		}
	}
	return nil
}

// loadFonts 加载多字体回退链：CJK 字体在前（覆盖中文），Latin 字体在后（覆盖 ASCII/数字/符号）。
// 多数系统 CJK 字体缺 ASCII、Latin 字体缺中文，必须混排（逐字符选字体）才能避免方块字。
func loadFonts() ([]*opentype.Font, error) {
	var fonts []*opentype.Font
	if f := loadFont(cjkFontCandidates); f != nil {
		fonts = append(fonts, f)
	}
	if f := loadFont(latinFontCandidates); f != nil {
		fonts = append(fonts, f)
	}
	if len(fonts) == 0 {
		return nil, fmt.Errorf("未找到可用字体（已尝试 %d 个系统路径）；请安装任意 CJK 或 Latin 字体", len(cjkFontCandidates)+len(latinFontCandidates))
	}
	return fonts, nil
}

// painter 长图画布（多字体回退混排）
type painter struct {
	img    *image.RGBA
	fonts  []*opentype.Font // 回退链：CJK 在前，Latin 在后
	faces  map[*opentype.Font]map[float64]font.Face
	buf    sfnt.Buffer
	main   *opentype.Font // 主字体（行高/基线基准，取第一个 CJK）
	W      int            // 画布宽
	maxW   int            // 内容最大宽
}

func newPainter(fonts []*opentype.Font, width, pad int) *painter {
	main := fonts[0]
	return &painter{
		img:   image.NewRGBA(image.Rect(0, 0, width, 9000)),
		fonts: fonts,
		faces: map[*opentype.Font]map[float64]font.Face{},
		main:  main,
		W:     width,
		maxW:  width - 2*pad,
	}
}

// face 按字体+字号缓存
func (p *painter) face(f *opentype.Font, size float64) font.Face {
	m, ok := p.faces[f]
	if !ok {
		m = map[float64]font.Face{}
		p.faces[f] = m
	}
	if ff, ok := m[size]; ok {
		return ff
	}
	ff, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		panic(err)
	}
	m[size] = ff
	return ff
}

// hasGlyph 判断字体是否含某字符的真实字形（g!=0 排除 .notdef 缺字形）
func (p *painter) hasGlyph(f *opentype.Font, r rune) bool {
	g, err := f.GlyphIndex(&p.buf, r)
	return err == nil && g != 0
}

// faceFor 返回渲染 rune 的字体（回退链中第一个含该字符字形的）
func (p *painter) faceFor(r rune, size float64) font.Face {
	for _, f := range p.fonts {
		if p.hasGlyph(f, r) {
			return p.face(f, size)
		}
	}
	return p.face(p.main, size)
}

func (p *painter) lineHeight(size float64) int {
	return p.face(p.main, size).Metrics().Height.Ceil()
}

func (p *painter) textWidth(text string, size float64) int {
	w := 0
	for _, r := range text {
		w += font.MeasureString(p.faceFor(r, size), string(r)).Ceil()
	}
	return w
}

// wrap 按最大宽度换行（中文按字符断行，逐字符测量）
func (p *painter) wrap(text string, size float64, maxW int) []string {
	var lines []string
	cur := ""
	for _, r := range text {
		t := cur + string(r)
		if cur != "" && p.textWidth(t, size) > maxW {
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

// text 绘制文本（自动换行，逐字符按字形选择字体），返回下一个可用 y（下一行顶部）
func (p *painter) text(text string, x, y int, size float64, col color.RGBA, maxW int) int {
	yy := y
	ascent := p.face(p.main, size).Metrics().Ascent.Ceil()
	for _, ln := range p.wrap(text, size, maxW) {
		xx := x
		baseline := yy + ascent
		for _, r := range ln {
			f := p.faceFor(r, size)
			d := &font.Drawer{Dst: p.img, Src: image.NewUniform(col), Face: f}
			d.Dot = fixed.P(xx, baseline)
			d.DrawString(string(r))
			xx += font.MeasureString(f, string(r)).Ceil()
		}
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
	fonts, err := loadFonts()
	if err != nil {
		return err
	}
	const (
		W   = 840
		pad = 48
	)
	p := newPainter(fonts, W, pad)
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
			meta = append(meta, "厨具 "+strings.TrimSpace(k))
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
			y = p.text(name, x, y, 20, colTitle, p.maxW) + 6
		}

		// 统计
		if v.Stats.PrepTime > 0 || v.Stats.CookTime > 0 {
			total := v.Stats.PrepTime + v.Stats.CookTime
			stat := fmt.Sprintf("准备 %d 分钟 · 烹饪 %d 分钟 · 总耗时 %d 分钟", v.Stats.PrepTime, v.Stats.CookTime, total)
			if v.Stats.Difficulty > 0 && v.Stats.Difficulty <= 5 {
				stat += fmt.Sprintf(" · 难度 %s", strings.Repeat("★", v.Stats.Difficulty))
			}
			y = p.text(stat, x, y, 15, colMuted, p.maxW) + 8
		}

		// 主要食材
		if len(v.Ingredients.Main) > 0 {
			y = p.text("主要食材", x, y, 21, colAccent, p.maxW) + 2
			for _, ing := range v.Ingredients.Main {
				y = p.text("· "+shareIngredient(ing), x+16, y, 16, colBody, p.maxW-16)
			}
			y += 6
		}
		// 配菜 / 辅料
		if len(v.Ingredients.Side) > 0 {
			y = p.text("配菜 / 辅料", x, y, 21, colAccent, p.maxW) + 2
			for _, ing := range v.Ingredients.Side {
				y = p.text("· "+shareIngredient(ing), x+16, y, 16, colBody, p.maxW-16)
			}
			y += 6
		}
		// 调料
		if len(v.Seasonings) > 0 {
			y = p.text("调料", x, y, 21, colAccent, p.maxW) + 2
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
			y = p.text("步骤", x, y, 21, colAccent, p.maxW) + 2
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
		y = p.text("完整菜谱："+siteRoot+"/recipe/"+r.ID, x, y, 13, colMuted, p.maxW)
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
