package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheManagerLifecycle(t *testing.T) {
	dir := t.TempDir()
	cm := NewCacheManager(filepath.Join(dir, "cache"))

	deps := map[string]string{"a.json": "hash1", "config": "cfg1"}

	// 初始无缓存
	if cm.IsValid("plugin1", deps, 86400) {
		t.Fatal("cache should be invalid before Save")
	}

	// 保存并验证有效
	if err := cm.SaveWithTTL("plugin1", []byte("hello"), deps, 86400); err != nil {
		t.Fatal(err)
	}
	if !cm.IsValid("plugin1", deps, 86400) {
		t.Fatal("cache should be valid after Save")
	}

	// 加载数据
	data, err := cm.Load("plugin1")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("loaded data = %q, want %q", data, "hello")
	}

	// 依赖变化导致失效
	if cm.IsValid("plugin1", map[string]string{"a.json": "changed"}, 86400) {
		t.Fatal("cache should be invalid when deps changed")
	}
}

func TestCacheTTL(t *testing.T) {
	dir := t.TempDir()
	cm := NewCacheManager(filepath.Join(dir, "cache"))
	deps := map[string]string{"a": "1"}

	if err := cm.SaveWithTTL("p", []byte("x"), deps, 1); err != nil {
		t.Fatal(err)
	}
	// 等待 TTL 过期
	time.Sleep(1100 * time.Millisecond)
	if cm.IsValid("p", deps, 1) {
		t.Fatal("cache should expire after TTL")
	}
	// 但更长 TTL 下仍有效（按 meta 记录的时间判断）
	if !cm.IsValid("p", deps, 3600) {
		t.Fatal("cache should be valid under longer TTL")
	}
}

func TestCacheClear(t *testing.T) {
	dir := t.TempDir()
	cm := NewCacheManager(filepath.Join(dir, "cache"))
	deps := map[string]string{"a": "1"}
	if err := cm.Save("p", []byte("x"), deps); err != nil {
		t.Fatal(err)
	}
	if err := cm.Clear("p"); err != nil {
		t.Fatal(err)
	}
	if cm.IsValid("p", deps, 86400) {
		t.Fatal("cache should be invalid after Clear")
	}
	if _, err := os.Stat(filepath.Join(dir, "cache", "p")); !os.IsNotExist(err) {
		t.Fatal("plugin dir should be removed after Clear")
	}
}

func TestSaveDefaultTTL(t *testing.T) {
	dir := t.TempDir()
	cm := NewCacheManager(dir)
	deps := map[string]string{"a": "1"}
	if err := cm.Save("p", []byte("x"), deps); err != nil {
		t.Fatal(err)
	}
	if !cm.IsValid("p", deps, 86400) {
		t.Fatal("default TTL save should be valid")
	}
}
