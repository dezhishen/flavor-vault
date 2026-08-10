package github

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DefaultIgnores 默认忽略的目录/文件（避免把本地缓存/构建产物/锁推送到远端）
var DefaultIgnores = []string{
	".git",
	".flavor-vault/cache",
	".flavor-vault/push.lock",
	"web/node_modules",
	"web/dist",
	"dist",
}

// CollectFiles 递归收集目录下文件，返回 相对路径 -> 内容。
// ignore 中每个条目按目录/文件前缀匹配（精确到段边界）。
func CollectFiles(root string, ignore []string) (map[string][]byte, error) {
	ignore = append(append([]string{}, DefaultIgnores...), ignore...)
	files := make(map[string][]byte)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if ignored(rel, ignore) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[rel] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("收集文件失败: %w", err)
	}
	return files, nil
}

// ignored 判断相对路径是否命中忽略规则（段边界匹配）
func ignored(rel string, ignore []string) bool {
	for _, ig := range ignore {
		ig = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(ig), "/"), "/")
		if ig == "" {
			continue
		}
		if rel == ig || strings.HasPrefix(rel, ig+"/") {
			return true
		}
	}
	return false
}

// SortedKeys 返回文件路径的有序列表（保证推送内容确定性）
func SortedKeys(files map[string][]byte) []string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
