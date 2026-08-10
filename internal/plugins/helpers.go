package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"flavor-vault/internal/pipeline"
	"flavor-vault/internal/utils"
)

// computeDeps 计算插件依赖哈希（菜谱文件 + config）
func computeDeps(ctx *pipeline.BuildContext, extra map[string]string) map[string]string {
	deps := make(map[string]string)
	for _, r := range ctx.Recipes {
		if r.FilePath != "" {
			deps[r.FilePath] = r.Hash
		}
	}
	if ctx.ConfigPath != "" {
		if h, err := utils.FileHash(ctx.ConfigPath); err == nil {
			deps["config:"+ctx.ConfigPath] = h
		}
	}
	for k, v := range extra {
		deps[k] = v
	}
	return deps
}

// cachedWrite 插件缓存集成模式：
// 1. 缓存有效时直接复用缓存数据写入输出
// 2. 无效时执行 build 并写入缓存与输出
func cachedWrite(ctx *pipeline.BuildContext, pluginName string, outPath string, build func() ([]byte, error)) error {
	ttl := ctx.Config.PluginTTL(pluginName)
	deps := computeDeps(ctx, nil)

	useCache := ctx.Config.CacheEnabled() && !ctx.Force
	if useCache && ctx.Cache.IsValid(pluginName, deps, ttl) {
		if data, err := ctx.Cache.Load(pluginName); err == nil {
			return pipeline.WriteFile(outPath, data)
		}
		// 缓存损坏，降级到全量重建
	}

	data, err := build()
	if err != nil {
		return err
	}
	if ctx.Config.CacheEnabled() {
		if err := ctx.Cache.SaveWithTTL(pluginName, data, deps, ttl); err != nil {
			return fmt.Errorf("保存缓存失败: %w", err)
		}
	}
	return pipeline.WriteFile(outPath, data)
}

// cachedWriteFiles 缓存一组文件输出（用于 tag_indexer、detail_splitter 等多文件插件）
// build 返回相对文件名 -> 内容
func cachedWriteFiles(ctx *pipeline.BuildContext, pluginName string, outDir string, build func() (map[string][]byte, error)) error {
	ttl := ctx.Config.PluginTTL(pluginName)
	deps := computeDeps(ctx, nil)
	useCache := ctx.Config.CacheEnabled() && !ctx.Force

	if useCache && ctx.Cache.IsValid(pluginName, deps, ttl) {
		if data, err := ctx.Cache.Load(pluginName); err == nil {
			var files map[string][]byte
			if json.Unmarshal(data, &files) == nil {
				for name, content := range files {
					if err := pipeline.WriteFile(filepath.Join(outDir, name), content); err != nil {
						return err
					}
				}
				return nil
			}
		}
		// 缓存损坏，降级到全量重建
	}

	files, err := build()
	if err != nil {
		return err
	}
	if ctx.Config.CacheEnabled() {
		data, _ := json.Marshal(files)
		if err := ctx.Cache.SaveWithTTL(pluginName, data, deps, ttl); err != nil {
			return fmt.Errorf("保存缓存失败: %w", err)
		}
	}
	for name, content := range files {
		if err := pipeline.WriteFile(filepath.Join(outDir, name), content); err != nil {
			return err
		}
	}
	return nil
}

// loadMeta 从文件加载统计信息
func loadMeta(path string) (*Meta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("加载统计信息失败（请先运行 fv build）: %w", err)
	}
	var m Meta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
