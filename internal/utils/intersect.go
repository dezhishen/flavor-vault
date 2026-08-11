package utils

import (
	"sort"
	"strings"
)

// IntersectSorted 计算两个有序字符串切片的交集（要求 a、b 均已排序）
// 返回的新切片保持有序。
func IntersectSorted(a, b []string) []string {
	result := make([]string, 0)
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			// 去重
			if len(result) == 0 || result[len(result)-1] != a[i] {
				result = append(result, a[i])
			}
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return result
}

// Intersect 计算多个有序字符串切片的交集。
// 返回所有输入共有的元素（有序）。若输入为空，返回空切片。
func Intersect(lists ...[]string) []string {
	if len(lists) == 0 {
		return []string{}
	}
	result := lists[0]
	for _, l := range lists[1:] {
		result = IntersectSorted(result, l)
		if len(result) == 0 {
			break
		}
	}
	return result
}

// SortAndDedupe 排序并去重
func SortAndDedupe(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	sort.Strings(items)
	out := items[:1]
	for _, s := range items[1:] {
		if out[len(out)-1] != s {
			out = append(out, s)
		}
	}
	return out
}

// Contains 判断切片是否包含某元素
func Contains(items []string, target string) bool {
	for _, s := range items {
		if s == target {
			return true
		}
	}
	return false
}

// IsRemoteURL 判断路径是否为外部资源（http/https/data: 协议）
func IsRemoteURL(p string) bool {
	l := strings.ToLower(strings.TrimSpace(p))
	return strings.HasPrefix(l, "http://") ||
		strings.HasPrefix(l, "https://") ||
		strings.HasPrefix(l, "data:")
}
