package cli

import (
	_ "embed"

	"golang.org/x/image/font/opentype"
)

// 内置字体（go:embed 打包进 fv 二进制，作为系统字体缺失时的兜底，保证任何环境都能渲染分享长图）
//
// - DroidSansFallbackFull.ttf：Google Droid Sans Fallback（Apache License 2.0），全量 CJK（中文）
// - DejaVuSans.ttf：DejaVu Sans（Bitstream Vera 衍生，可自由分发），ASCII/数字/符号（含 ★·）
//
// 两者互补：Droid 缺 ASCII、DejaVu 缺中文；配合 img.go 的多字体回退混排逐字符选字体，覆盖全部字符。
//
//go:embed fonts/DroidSansFallbackFull.ttf
var embeddedCJK []byte

//go:embed fonts/DejaVuSans.ttf
var embeddedLatin []byte

// parseEmbedded 从内置字节解析字体（ttf/ttc 均可）
func parseEmbedded(data []byte) *opentype.Font {
	if f, err := opentype.Parse(data); err == nil {
		return f
	}
	if coll, err := opentype.ParseCollection(data); err == nil && coll.NumFonts() > 0 {
		if f, err := coll.Font(0); err == nil {
			return f
		}
	}
	return nil
}
