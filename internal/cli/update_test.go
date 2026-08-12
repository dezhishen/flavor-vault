package cli

import "testing"

// TestIsNewerPre 验证 isNewer 对预发布版本的比较（beta.5 > beta.1，正式 > 预发布）
func TestIsNewerPre(t *testing.T) {
	cases := []struct {
		cur, target string
		want        bool
	}{
		// 数字相同，预发布序号递增
		{"0.0.3-beta.1", "0.0.3-beta.5", true},
		{"0.0.3-beta.5", "0.0.3-beta.1", false},
		{"0.0.3-beta.1", "0.0.3-beta.10", true}, // 数字段按数值比较
		{"0.0.3-beta.10", "0.0.3-beta.2", false},
		// 相同版本 → 不更新
		{"0.0.3-beta.5", "0.0.3-beta.5", false},
		// 正式版 > 预发布
		{"0.0.3-beta.5", "0.0.3", true},
		{"0.0.3", "0.0.3-beta.1", false},
		// 主版本号不同优先于预发布
		{"0.0.3-beta.9", "0.0.4-alpha.1", true},
		{"0.0.3", "0.1.0-beta.1", true},
		// 前缀相同时段数更多更新（beta < beta.1）
		{"0.0.3-beta", "0.0.3-beta.1", true},
		{"0.0.3-beta.1", "0.0.3-beta", false},
	}
	for _, c := range cases {
		if got := isNewer(c.cur, c.target); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.cur, c.target, got, c.want)
		}
	}
}

// TestPreGreater 验证预发布段比较
func TestPreGreater(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"beta.5", "beta.1", true},
		{"beta.1", "beta.5", false},
		{"beta.10", "beta.2", true},
		{"beta.1", "beta.1", false},
		{"alpha.1", "beta.1", false}, // 字典序 alpha < beta
		{"beta", "beta.1", false},    // 缺失段更小
		{"beta.1", "beta", true},
		{"rc.1", "beta.9", true},
	}
	for _, c := range cases {
		if got := preGreater(c.a, c.b); got != c.want {
			t.Errorf("preGreater(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
