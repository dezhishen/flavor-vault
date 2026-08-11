package models

import "strings"
import "time"

// Recipe 菜谱模型
type Recipe struct {
	ID          string      `json:"id"`           // 唯一标识（建议拼音或 UUID）
	Name        string      `json:"name"`         // 菜名
	Description string      `json:"description"`  // 简介
	Tags        []string    `json:"tags"`         // 标签（如 "凉菜","川菜"）
	Kitchenware []string    `json:"kitchenware"`  // 厨具（如 "炒锅","砂锅"）
	Ingredients Ingredients `json:"ingredients"`
	Steps       []Step      `json:"steps"`
	Media       Media       `json:"media"`
	Stats       Stats       `json:"stats"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	// 内部使用，不序列化
	FilePath string `json:"-"` // 源文件路径
	Hash     string `json:"-"` // 源文件内容哈希（构建时使用）
}

// Ingredients 食材分组
type Ingredients struct {
	Main []Ingredient `json:"main"`
	Side []Ingredient `json:"side"` // 配菜/辅料
}

// Ingredient 单个食材
type Ingredient struct {
	Name   string `json:"name"`
	Amount string `json:"amount"` // 如 "500g", "适量"
}

// Step 步骤
type Step struct {
	Order       int    `json:"order"`
	Description string `json:"description"`
	ImageRef    string `json:"image_ref,omitempty"` // 步骤图（可选）
}

// Media 媒体资源
type Media struct {
	Cover    string   `json:"cover"`               // 封面图路径
	Images   []string `json:"images"`              // 过程图列表
	VideoURL string   `json:"video_url,omitempty"` // 外部视频链接
}

// Stats 统计信息
type Stats struct {
	PrepTime   int `json:"prep_time"`   // 准备分钟
	CookTime   int `json:"cook_time"`   // 烹饪分钟
	Difficulty int `json:"difficulty"`  // 1-5
}

// MainIngredientNames 返回主要食材名称列表（用于索引）
func (r *Recipe) MainIngredientNames() []string {
	names := make([]string, 0, len(r.Ingredients.Main))
	for _, ing := range r.Ingredients.Main {
		names = append(names, ing.Name)
	}
	return names
}

// TotalTime 总耗时（分钟）
func (r *Recipe) TotalTime() int {
	return r.Stats.PrepTime + r.Stats.CookTime
}

// AssetRefs 返回该菜谱引用的所有资源路径（封面/过程图/步骤图，去重、含外部 URL）
func (r *Recipe) AssetRefs() []string {
	seen := make(map[string]bool)
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	add(r.Media.Cover)
	for _, img := range r.Media.Images {
		add(img)
	}
	for _, s := range r.Steps {
		add(s.ImageRef)
	}
	return out
}
