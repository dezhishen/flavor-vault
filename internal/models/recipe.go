package models

import (
	"encoding/json"
	"strings"
	"time"
)

// Recipe 菜谱模型
type Recipe struct {
	ID          string      `json:"id"`                   // 唯一标识（建议拼音或 UUID）
	Name        string      `json:"name"`                 // 菜名
	Description string      `json:"description"`          // 简介
	Tags        []string    `json:"tags"`                 // 标签（如 "凉菜","川菜"）
	Kitchenware []string    `json:"kitchenware"`          // 厨具（如 "炒锅","砂锅"）
	Ingredients Ingredients `json:"ingredients"`          // 默认版本食材
	Seasonings  []Seasoning `json:"seasonings,omitempty"` // 默认版本调料（可含备选方案）
	Steps       []Step      `json:"steps"`                // 默认版本步骤
	Media       Media       `json:"media"`                // 默认版本媒体
	Stats       Stats       `json:"stats"`                // 默认版本统计
	Versions    []Version   `json:"versions,omitempty"`   // 多版本（非空时优先使用；为空则用顶层字段作为默认版本）
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	// 内部使用，不序列化
	FilePath string `json:"-"` // 源文件路径
	Hash     string `json:"-"` // 源文件内容哈希（构建时使用）
}

// Ingredients 食材分组
//   - Main：主料（必选）
//   - Side：配菜/辅料
// 非必须（可选）通过条目 Note 备注表达（如 "可省略"），不设独立分组。

type Ingredients struct {
	Main []Ingredient `json:"main"`
	Side []Ingredient `json:"side"`
}

// Ingredient 单个食材
// 必选/非必须由 Note 备注表达（如 "可省略"），可替换方案由 Alternatives 表达。
type Ingredient struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`         // 如 "500g", "适量"
	Note   string `json:"note,omitempty"` // 备注（如 "没有可省略"）
	// 可替换食材（方案二/三…，如 用梅花肉代替五花肉）
	Alternatives []IngredientOption `json:"alternatives,omitempty"`
}

// IngredientOption 食材可替换方案
// （如 梅花肉 代替 五花肉；与 SeasoningOption 结构平行）
type IngredientOption struct {
	Name   string `json:"name"`             // 如 "梅花肉"
	Amount string `json:"amount,omitempty"` // 用量
	Note   string `json:"note,omitempty"`   // 如 "代替五花肉"
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
	PrepTime   int `json:"prep_time"`  // 准备分钟
	CookTime   int `json:"cook_time"`  // 烹饪分钟
	Difficulty int `json:"difficulty"` // 1-5
}

// Seasoning 调料条目（方案一），可带备选方案（方案二/三…，如用香菜代替香葱）
type Seasoning struct {
	Name         string            `json:"name"`                   // 方案一名称（如 "香葱"）
	Amount       string            `json:"amount,omitempty"`       // 用量
	Note         string            `json:"note,omitempty"`         // 备注
	Alternatives []SeasoningOption `json:"alternatives,omitempty"` // 备选方案（方案二/三…）
}

// SeasoningOption 调料备选方案
type SeasoningOption struct {
	Name   string `json:"name"`             // 如 "香菜"
	Amount string `json:"amount,omitempty"` // 用量
	Note   string `json:"note,omitempty"`   // 如 "代替香葱"
}

// Version 菜谱版本（同一道菜的不同做法/口味/规格）
type Version struct {
	Name        string      `json:"name"` // 版本名（如 "经典版" / "免辣版"）
	Description string      `json:"description,omitempty"`
	Ingredients Ingredients `json:"ingredients"`
	Seasonings  []Seasoning `json:"seasonings,omitempty"`
	Steps       []Step      `json:"steps"`
	Media       Media       `json:"media"`
	Stats       Stats       `json:"stats"`
}

// VersionsEffective 返回有效版本列表：versions 非空时用 versions；
// 否则把顶层字段（ingredients/seasonings/steps/media/stats）作为单个默认版本。
func (r *Recipe) VersionsEffective() []Version {
	if len(r.Versions) > 0 {
		return r.Versions
	}
	return []Version{{
		Ingredients: r.Ingredients,
		Seasonings:  r.Seasonings,
		Steps:       r.Steps,
		Media:       r.Media,
		Stats:       r.Stats,
	}}
}

// MainIngredientNames 聚合所有版本的“主要食材”名称（去重保序，用于倒排索引/摘要）
func (r *Recipe) MainIngredientNames() []string {
	var out []string
	seen := make(map[string]bool)
	for _, v := range r.VersionsEffective() {
		for _, ing := range v.Ingredients.Main {
			n := strings.TrimSpace(ing.Name)
			if n == "" || seen[n] {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// IngredientNamesAll 聚合所有版本的食材（主/配/可选）与调料（含备选）名称（用于搜索索引）
func (r *Recipe) IngredientNamesAll() []string {
	var out []string
	seen := make(map[string]bool)
	add := func(n string) {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			return
		}
		seen[n] = true
		out = append(out, n)
	}
	for _, v := range r.VersionsEffective() {
		groups := append([]Ingredient{}, v.Ingredients.Main...)
		groups = append(groups, v.Ingredients.Side...)
		for _, ing := range groups {
			add(ing.Name)
			for _, alt := range ing.Alternatives {
				add(alt.Name)
			}
		}
		for _, s := range v.Seasonings {
			add(s.Name)
			for _, alt := range s.Alternatives {
				add(alt.Name)
			}
		}
	}
	return out
}

// MarshalJSON 空切片统一序列化为 []，避免输出 null（前端消费更安全）
func (r Recipe) MarshalJSON() ([]byte, error) {
	type Alias Recipe
	return json.Marshal(&struct {
		*Alias
		Tags        []string    `json:"tags"`
		Kitchenware []string    `json:"kitchenware"`
		Ingredients Ingredients `json:"ingredients"`
		Seasonings  []Seasoning `json:"seasonings"`
		Steps       []Step      `json:"steps"`
		Media       Media       `json:"media"`
		Versions    []Version   `json:"versions"`
	}{
		Alias:       (*Alias)(&r),
		Tags:        nonNilStrings(r.Tags),
		Kitchenware: nonNilStrings(r.Kitchenware),
		Ingredients: r.Ingredients.normalized(),
		Seasonings:  nonNilSeasonings(r.Seasonings),
		Steps:       nonNilSteps(r.Steps),
		Media:       r.Media.normalized(),
		Versions:    nonNilVersions(r.Versions),
	})
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func nonNilSteps(s []Step) []Step {
	if s == nil {
		return []Step{}
	}
	return s
}

func nonNilSeasonings(s []Seasoning) []Seasoning {
	if s == nil {
		return []Seasoning{}
	}
	return s
}

func nonNilVersions(s []Version) []Version {
	if s == nil {
		return []Version{}
	}
	out := make([]Version, len(s))
	for i := range s {
		out[i] = s[i].normalized()
	}
	return out
}

func (in Ingredients) normalized() Ingredients {
	if in.Main == nil {
		in.Main = []Ingredient{}
	}
	if in.Side == nil {
		in.Side = []Ingredient{}
	}
	return in
}

func (v Version) normalized() Version {
	v.Ingredients = v.Ingredients.normalized()
	v.Seasonings = nonNilSeasonings(v.Seasonings)
	v.Steps = nonNilSteps(v.Steps)
	v.Media = v.Media.normalized()
	return v
}

func (m Media) normalized() Media {
	if m.Images == nil {
		m.Images = []string{}
	}
	return m
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
	for _, v := range r.VersionsEffective() {
		add(v.Media.Cover)
		for _, img := range v.Media.Images {
			add(img)
		}
		for _, s := range v.Steps {
			add(s.ImageRef)
		}
	}
	return out
}
