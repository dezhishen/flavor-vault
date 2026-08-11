package vault

import (
	"os"
	"path/filepath"
	"testing"

	"flavor-vault/internal/models"
)

func TestResolveWithConfigStandardLayout(t *testing.T) {
	// <root>/.flavor-vault/config.yaml → root = <root>
	base := t.TempDir()
	cfg := filepath.Join(base, "myapp", DirName, ConfigName)
	root, resolved, err := ResolveWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if root != filepath.Join(base, "myapp") {
		t.Errorf("root = %s, want %s", root, filepath.Join(base, "myapp"))
	}
	if resolved != cfg {
		t.Errorf("resolved = %s, want %s", resolved, cfg)
	}
}

func TestResolveWithConfigCustomPath(t *testing.T) {
	// 自定义路径 → root = 所在目录
	base := t.TempDir()
	cfg := filepath.Join(base, "vault", "custom.yaml")
	root, resolved, err := ResolveWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if root != filepath.Join(base, "vault") {
		t.Errorf("root = %s, want %s", root, filepath.Join(base, "vault"))
	}
	if resolved != cfg {
		t.Errorf("resolved = %s, want %s", resolved, cfg)
	}
}

func TestSaveAndLoadConfigAt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conf", "config.yaml")

	cfg := models.DefaultConfig()
	cfg.Endpoint = "https://example.com/data"
	cfg.OutputDir = "/custom/dist"

	if err := SaveConfigAt(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfigAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Endpoint != "https://example.com/data" {
		t.Errorf("endpoint = %s, want https://example.com/data", loaded.Endpoint)
	}
	if loaded.OutputDir != "/custom/dist" {
		t.Errorf("output_dir = %s, want /custom/dist", loaded.OutputDir)
	}
}

func TestLoadConfigAtMissingFileReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.yaml")
	cfg, err := LoadConfigAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OutputDir == "" {
		t.Error("default config should have output_dir")
	}
}

func TestResolveContextEmptyFlag(t *testing.T) {
	// 无 --config 时自动查找 .flavor-vault
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, DirName), 0o755); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	root, cfgPath, err := ResolveContext("")
	if err != nil {
		t.Fatal(err)
	}
	if root != dir {
		t.Errorf("root = %s, want %s", root, dir)
	}
	wantCfg := filepath.Join(dir, DirName, ConfigName)
	if cfgPath != wantCfg {
		t.Errorf("configPath = %s, want %s", cfgPath, wantCfg)
	}
}

func TestResolveContextCustomFlag(t *testing.T) {
	root, cfgPath, err := ResolveContext("/data/myvault/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if root != "/data/myvault" {
		t.Errorf("root = %s, want /data/myvault", root)
	}
	if cfgPath != "/data/myvault/config.yaml" {
		t.Errorf("configPath = %s", cfgPath)
	}
}

func TestResolveRecipesDir(t *testing.T) {
	dir := t.TempDir()

	// 无独立分支配置 → 默认目录
	cfg := models.DefaultConfig()
	if got := ResolveRecipesDir(dir, cfg); got != RecipesDir(dir) {
		t.Errorf("no source: got %s, want %s", got, RecipesDir(dir))
	}
	// cfg 为 nil 时同样默认
	if got := ResolveRecipesDir(dir, nil); got != RecipesDir(dir) {
		t.Errorf("nil cfg: got %s", got)
	}

	// 配置了读写（数据仓库）但 worktree 不存在 → 回退默认
	cfg.GitHub.Branch = "recipes"
	if got := ResolveRecipesDir(dir, cfg); got != RecipesDir(dir) {
		t.Errorf("worktree missing: got %s, want %s", got, RecipesDir(dir))
	}

	// 配置了读写（数据仓库）且 worktree 存在 → worktree 下的菜谱目录
	wt := RecipesWorktree(dir)
	if err := os.MkdirAll(filepath.Join(wt, DirName, RecipesDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(wt, DirName, RecipesDirName)
	if got := ResolveRecipesDir(dir, cfg); got != want {
		t.Errorf("worktree present: got %s, want %s", got, want)
	}
}
