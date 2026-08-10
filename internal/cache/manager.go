package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PluginMeta 单个插件的缓存元数据
type PluginMeta struct {
	GeneratedAt time.Time         `json:"generated_at"`
	TTLSeconds  int               `json:"ttl_seconds"`
	Deps        map[string]string `json:"deps"` // 依赖文件路径 -> 哈希
}

// Manifest 全局缓存索引（记录各插件缓存状态）
type Manifest map[string]ManifestEntry

// ManifestEntry 清单条目
type ManifestEntry struct {
	GeneratedAt time.Time `json:"generated_at"`
	TTLSeconds  int       `json:"ttl_seconds"`
}

// CacheManager 缓存管理器
type CacheManager struct {
	rootDir string // .flavor-vault/cache
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(rootDir string) *CacheManager {
	return &CacheManager{rootDir: rootDir}
}

func (cm *CacheManager) pluginDir(pluginName string) string {
	return filepath.Join(cm.rootDir, pluginName)
}

func (cm *CacheManager) dataPath(pluginName string) string {
	return filepath.Join(cm.pluginDir(pluginName), "data.gob")
}

func (cm *CacheManager) metaPath(pluginName string) string {
	return filepath.Join(cm.pluginDir(pluginName), "meta.json")
}

func (cm *CacheManager) manifestPath() string {
	return filepath.Join(cm.rootDir, "manifest.json")
}

// IsValid 检查缓存是否有效：依赖哈希一致 && 未超过 TTL
func (cm *CacheManager) IsValid(pluginName string, deps map[string]string, ttlSeconds int) bool {
	if ttlSeconds <= 0 {
		return false
	}
	meta, err := cm.loadMeta(pluginName)
	if err != nil {
		return false
	}
	// 检查数据文件存在
	if _, err := os.Stat(cm.dataPath(pluginName)); err != nil {
		return false
	}
	// 检查依赖哈希一致
	if !sameDeps(meta.Deps, deps) {
		return false
	}
	// 检查 TTL（以当前传入的 TTL 为准）
	if time.Since(meta.GeneratedAt) > time.Duration(ttlSeconds)*time.Second {
		return false
	}
	return true
}

// Save 保存缓存数据
func (cm *CacheManager) Save(pluginName string, data []byte, deps map[string]string) error {
	dir := cm.pluginDir(pluginName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(cm.dataPath(pluginName), data, 0o644); err != nil {
		return err
	}
	meta := &PluginMeta{
		GeneratedAt: time.Now(),
		TTLSeconds:  86400, // 默认，可由插件覆盖
		Deps:        deps,
	}
	return cm.saveMeta(pluginName, meta)
}

// SaveWithTTL 保存缓存数据并指定 TTL
func (cm *CacheManager) SaveWithTTL(pluginName string, data []byte, deps map[string]string, ttlSeconds int) error {
	dir := cm.pluginDir(pluginName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(cm.dataPath(pluginName), data, 0o644); err != nil {
		return err
	}
	meta := &PluginMeta{
		GeneratedAt: time.Now(),
		TTLSeconds:  ttlSeconds,
		Deps:        deps,
	}
	return cm.saveMeta(pluginName, meta)
}

// Load 加载缓存数据
func (cm *CacheManager) Load(pluginName string) ([]byte, error) {
	data, err := os.ReadFile(cm.dataPath(pluginName))
	if err != nil {
		return nil, fmt.Errorf("读取缓存 %s 失败: %w", pluginName, err)
	}
	return data, nil
}

// Clear 强制清除某个插件的缓存
func (cm *CacheManager) Clear(pluginName string) error {
	dir := cm.pluginDir(pluginName)
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	// 同步更新 manifest
	manifest, err := cm.loadManifest()
	if err != nil {
		return nil // manifest 不存在则忽略
	}
	delete(manifest, pluginName)
	return cm.saveManifest(manifest)
}

// ClearAll 清除所有缓存
func (cm *CacheManager) ClearAll() error {
	return os.RemoveAll(cm.rootDir)
}

// ---------------------------------------------------------------------------

func (cm *CacheManager) loadMeta(pluginName string) (*PluginMeta, error) {
	data, err := os.ReadFile(cm.metaPath(pluginName))
	if err != nil {
		return nil, err
	}
	var meta PluginMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (cm *CacheManager) saveMeta(pluginName string, meta *PluginMeta) error {
	if err := os.MkdirAll(cm.pluginDir(pluginName), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(cm.metaPath(pluginName), data, 0o644); err != nil {
		return err
	}

	// 更新全局 manifest
	manifest, err := cm.loadManifest()
	if err != nil {
		manifest = make(Manifest)
	}
	manifest[pluginName] = ManifestEntry{
		GeneratedAt: meta.GeneratedAt,
		TTLSeconds:  meta.TTLSeconds,
	}
	return cm.saveManifest(manifest)
}

func (cm *CacheManager) loadManifest() (Manifest, error) {
	data, err := os.ReadFile(cm.manifestPath())
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (cm *CacheManager) saveManifest(m Manifest) error {
	if err := os.MkdirAll(cm.rootDir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cm.manifestPath(), data, 0o644)
}

func sameDeps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
