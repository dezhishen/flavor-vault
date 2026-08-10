package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"flavor-vault/internal/models"
)

// RecipeFileStore 单个菜谱的 CRUD（读写 JSON）
type RecipeFileStore struct {
	recipesDir string
}

// NewRecipeFileStore 创建存储
func NewRecipeFileStore(recipesDir string) *RecipeFileStore {
	return &RecipeFileStore{recipesDir: recipesDir}
}

// PathFor 返回某 ID 对应的文件路径
func (s *RecipeFileStore) PathFor(id string) string {
	return filepath.Join(s.recipesDir, id+".json")
}

// Exists 判断菜谱是否存在
func (s *RecipeFileStore) Exists(id string) bool {
	_, err := os.Stat(s.PathFor(id))
	return err == nil
}

// Save 写入菜谱（自动更新 UpdatedAt）
func (s *RecipeFileStore) Save(r *models.Recipe) error {
	if r.ID == "" {
		return fmt.Errorf("菜谱 ID 不能为空")
	}
	now := time.Now()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	r.UpdatedAt = now

	if err := os.MkdirAll(s.recipesDir, 0o755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	// 原子写入：先写临时文件再重命名
	tmp := s.PathFor(r.ID) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入失败: %w", err)
	}
	if err := os.Rename(tmp, s.PathFor(r.ID)); err != nil {
		return fmt.Errorf("重命名失败: %w", err)
	}
	return nil
}

// Load 读取单个菜谱
func (s *RecipeFileStore) Load(id string) (*models.Recipe, error) {
	path := s.PathFor(id)
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("菜谱 %q 不存在", id)
	}
	r, _, err := LoadOne(path)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// Delete 删除菜谱文件
func (s *RecipeFileStore) Delete(id string) error {
	path := s.PathFor(id)
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("菜谱 %q 不存在", id)
	}
	return os.Remove(path)
}
