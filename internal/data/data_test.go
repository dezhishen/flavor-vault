package data

import (
	"os"
	"path/filepath"
	"testing"

	"flavor-vault/internal/models"
)

func TestRemoteEndpoint(t *testing.T) {
	// 配置了 endpoint → 返回去尾斜杠后的地址
	consumer := &models.Config{Endpoint: "https://example.com/data/"}
	if got := RemoteEndpoint(consumer); got != "https://example.com/data" {
		t.Errorf("endpoint = %q, want trimmed", got)
	}
	// 未配置 → 空
	if got := RemoteEndpoint(&models.Config{}); got != "" {
		t.Errorf("empty endpoint = %q, want empty", got)
	}
	// 读取只依赖 endpoint（即使维护者配置了 source 也照常返回）
	m := &models.Config{Endpoint: "https://example.com/data", GitHub: models.GitHubConfig{Branch: "recipes"}}
	if got := RemoteEndpoint(m); got != "https://example.com/data" {
		t.Errorf("maintainer endpoint = %q, want set", got)
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
	// 未配置 endpoint → 从本地 meta.json 取默认（构建时注入）
	if got := DefaultEndpointFromMeta(cfg, dir); got != "https://dezhishen.github.io/flavor-vault/data" {
		t.Errorf("DefaultEndpointFromMeta = %q", got)
	}
	// meta 缺失 → 空
	if got := DefaultEndpointFromMeta(cfg, t.TempDir()); got != "" {
		t.Errorf("no meta DefaultEndpointFromMeta = %q, want empty", got)
	}
	// 内置默认端点
	if DefaultEndpoint != "https://fv.sdniu.top/data" {
		t.Errorf("DefaultEndpoint const = %q", DefaultEndpoint)
	}
}
