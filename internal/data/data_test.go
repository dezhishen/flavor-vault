package data

import (
	"os"
	"path/filepath"
	"testing"

	"flavor-vault/internal/models"
)

func TestRemoteEndpointConsumerVsMaintainer(t *testing.T) {
	// 使用者模式：返回配置的 endpoint
	consumer := &models.Config{Endpoint: "https://example.com/data/"}
	if got := RemoteEndpoint(consumer); got != "https://example.com/data" {
		t.Errorf("consumer endpoint = %q, want trimmed", got)
	}
	// 使用者模式：未配置 → 空
	if got := RemoteEndpoint(&models.Config{}); got != "" {
		t.Errorf("consumer empty endpoint = %q, want empty", got)
	}
	// 维护者模式：即便配置了 endpoint 也不使用（本地数据源为准）
	m := &models.Config{Endpoint: "https://example.com/data", Source: models.SourceConfig{Branch: "recipes"}}
	if got := RemoteEndpoint(m); got != "" {
		t.Errorf("maintainer endpoint = %q, want empty", got)
	}
}

func TestDefaultEndpointFromMeta(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "dist")
	if err := os.MkdirAll(filepath.Join(out, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	meta := `{"total": 3, "endpoint": "https://dezhishen.github.io/flavor-vault/data"}`
	if err := os.WriteFile(filepath.Join(out, "data", "meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &models.Config{OutputDir: "./dist"}
	// 使用者模式：未配置 endpoint → 从本地 meta.json 取默认
	if got := DefaultEndpoint(cfg, dir); got != "https://dezhishen.github.io/flavor-vault/data" {
		t.Errorf("DefaultEndpoint = %q", got)
	}
	// 维护者模式：不使用默认 endpoint
	m := &models.Config{OutputDir: "./dist", Source: models.SourceConfig{Branch: "recipes"}}
	if got := DefaultEndpoint(m, dir); got != "" {
		t.Errorf("maintainer DefaultEndpoint = %q, want empty", got)
	}
	// meta 缺失 → 空
	if got := DefaultEndpoint(cfg, t.TempDir()); got != "" {
		t.Errorf("no meta DefaultEndpoint = %q, want empty", got)
	}
}
