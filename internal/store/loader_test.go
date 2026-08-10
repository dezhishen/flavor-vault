package store

import (
	"os"
	"path/filepath"
	"testing"

	"flavor-vault/internal/models"
)

func writeTestRecipe(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const validRecipe = `{
  "id": "test-1",
  "name": "测试菜",
  "tags": ["家常"],
  "kitchenware": ["炒锅"],
  "ingredients": {"main": [{"name": "土豆", "amount": "2个"}]},
  "steps": [{"order": 1, "description": "第一步"}],
  "stats": {"prep_time": 5, "cook_time": 10, "difficulty": 2}
}`

func TestLoadAll(t *testing.T) {
	dir := t.TempDir()
	writeTestRecipe(t, dir, "test-1.json", validRecipe)
	writeTestRecipe(t, dir, "bad.json", "{ not valid json")
	writeTestRecipe(t, dir, "ignore.txt", "not a recipe")

	// SkipInvalid=true 时应跳过坏文件并给出警告
	res, err := LoadAll(dir, LoadOptions{SkipInvalid: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Recipes) != 1 {
		t.Fatalf("expected 1 recipe, got %d", len(res.Recipes))
	}
	if len(res.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(res.Warnings), res.Warnings)
	}
	if res.Recipes[0].Name != "测试菜" {
		t.Errorf("unexpected recipe: %+v", res.Recipes[0])
	}
	if res.Recipes[0].Hash == "" {
		t.Error("recipe hash should be set")
	}

	// SkipInvalid=false 时应返回错误
	if _, err := LoadAll(dir, LoadOptions{SkipInvalid: false}); err == nil {
		t.Fatal("expected error for invalid recipe file")
	}
}

func TestRecipeFileStoreCRUD(t *testing.T) {
	dir := t.TempDir()
	fs := NewRecipeFileStore(dir)

	r := &models.Recipe{ID: "hong-shao-rou", Name: "红烧肉"}
	if err := fs.Save(r); err != nil {
		t.Fatal(err)
	}
	if !fs.Exists("hong-shao-rou") {
		t.Fatal("recipe should exist after Save")
	}
	if r.CreatedAt.IsZero() || r.UpdatedAt.IsZero() {
		t.Fatal("timestamps should be set")
	}

	loaded, err := fs.Load("hong-shao-rou")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != "红烧肉" {
		t.Errorf("unexpected loaded recipe: %+v", loaded)
	}

	if err := fs.Delete("hong-shao-rou"); err != nil {
		t.Fatal(err)
	}
	if fs.Exists("hong-shao-rou") {
		t.Fatal("recipe should be gone after Delete")
	}
	if _, err := fs.Load("hong-shao-rou"); err == nil {
		t.Fatal("Load should error for missing recipe")
	}
}
