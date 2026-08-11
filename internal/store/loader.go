package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"flavor-vault/internal/models"
	"flavor-vault/internal/utils"
)

// LoadOptions 加载选项
type LoadOptions struct {
	// SkipInvalid 为 true 时跳过无法解析的文件（并记录警告）
	SkipInvalid bool
}

// LoadResult 加载结果
type LoadResult struct {
	Recipes   []*models.Recipe
	Warnings  []string // 格式错误文件的警告
	RawHashes map[string]string // 文件路径 -> 哈希
}

// LoadAll 加载 recipes 目录下所有菜谱 JSON
func LoadAll(recipesDir string, opts LoadOptions) (*LoadResult, error) {
	res := &LoadResult{
		Recipes:   make([]*models.Recipe, 0),
		Warnings:  make([]string, 0),
		RawHashes: make(map[string]string),
	}

	entries, err := os.ReadDir(recipesDir)
	if err != nil {
		return nil, fmt.Errorf("读取菜谱目录失败: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		path := filepath.Join(recipesDir, e.Name())
		recipe, warn, err := LoadOne(path)
		if err != nil {
			if opts.SkipInvalid {
				res.Warnings = append(res.Warnings, fmt.Sprintf("跳过 %s: %v", path, err))
				continue
			}
			return nil, err
		}
		if warn != "" {
			res.Warnings = append(res.Warnings, warn)
		}
		hash, _ := utils.FileHash(path)
		recipe.Hash = hash
		res.RawHashes[path] = hash
		res.Recipes = append(res.Recipes, recipe)
	}

	// 按 ID 排序保证确定性
	sort.Slice(res.Recipes, func(i, j int) bool {
		return res.Recipes[i].ID < res.Recipes[j].ID
	})
	return res, nil
}

// LoadAllMulti 从多个菜谱目录加载（本地 + 外部数据源），跳过不存在的目录。
// 用于维护者聚合自有菜谱与引用的外部仓库菜谱。
func LoadAllMulti(dirs []string, opts LoadOptions) (*LoadResult, error) {
	res := &LoadResult{
		Recipes:   make([]*models.Recipe, 0),
		Warnings:  make([]string, 0),
		RawHashes: make(map[string]string),
	}
	for _, d := range dirs {
		if _, err := os.Stat(d); err != nil {
			continue // 目录不存在（如尚未 pull 的外部源）直接跳过
		}
		r, err := LoadAll(d, opts)
		if err != nil {
			return nil, err
		}
		res.Recipes = append(res.Recipes, r.Recipes...)
		res.Warnings = append(res.Warnings, r.Warnings...)
		for k, v := range r.RawHashes {
			res.RawHashes[k] = v
		}
	}
	sort.Slice(res.Recipes, func(i, j int) bool {
		return res.Recipes[i].ID < res.Recipes[j].ID
	})
	return res, nil
}

// LoadOne 加载单个菜谱文件，返回 (recipe, warning, error)
func LoadOne(path string) (*models.Recipe, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var r models.Recipe
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, "", fmt.Errorf("JSON 解析失败: %w", err)
	}
	if r.ID == "" {
		// 以文件名作为兜底 ID
		r.ID = trimExt(filepath.Base(path))
		warn := fmt.Sprintf("菜谱 %s 缺少 id 字段，已使用文件名作为 id", path)
		r.FilePath = path
		return &r, warn, nil
	}
	r.FilePath = path
	return &r, "", nil
}

func trimExt(name string) string {
	return name[:len(name)-len(filepath.Ext(name))]
}
