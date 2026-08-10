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
	cfg.Tags = append(cfg.Tags, "测试标签")
	cfg.OutputDir = "/custom/dist"

	if err := SaveConfigAt(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfigAt(path)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(loaded.Tags, "测试标签") {
		t.Errorf("tags = %v, want to contain 测试标签", loaded.Tags)
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

func contains(items []string, target string) bool {
	for _, s := range items {
		if s == target {
			return true
		}
	}
	return false
}
