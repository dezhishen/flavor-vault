package github

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectFilesAndIgnore(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("recipes/a.json", `{"id":"a"}`)
	write("recipes/sub/b.json", `{"id":"b"}`)
	write(".flavor-vault/config.yaml", "tags: []")
	write(".flavor-vault/cache/plugin/data.gob", "cache")
	write("web/node_modules/x/index.js", "node")
	write("dist/index.html", "html")
	write("README.md", "readme")

	files, err := CollectFiles(root, nil)
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{"recipes/a.json", "recipes/sub/b.json", ".flavor-vault/config.yaml", "README.md"} {
		if _, ok := files[p]; !ok {
			t.Errorf("expected file %s to be collected", p)
		}
	}
	for _, p := range []string{".flavor-vault/cache/plugin/data.gob", "web/node_modules/x/index.js", "dist/index.html"} {
		if _, ok := files[p]; ok {
			t.Errorf("file %s should be ignored (default ignore)", p)
		}
	}

	// 自定义 ignore
	files2, err := CollectFiles(root, []string{"recipes"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files2["recipes/a.json"]; ok {
		t.Error("recipes/a.json should be ignored by custom ignore")
	}
}

func TestIgnored(t *testing.T) {
	cases := []struct {
		rel, ignore string
		want        bool
	}{
		{"a/b.txt", "a", true},
		{"a/b/c.txt", "a", true},
		{"ab/c.txt", "a", false}, // 段边界：ab 不以 a 为独立目录
		{"a.txt", "a", false},
		{"dist/x.js", "dist", true},
		{"dist2/x.js", "dist", false},
		{"x/.gitignore", ".git", false}, // .gitignore 不是 .git 目录
	}
	for _, c := range cases {
		if got := ignored(c.rel, []string{c.ignore}); got != c.want {
			t.Errorf("ignored(%q, %q) = %v, want %v", c.rel, c.ignore, got, c.want)
		}
	}
}

func TestParseRemoteURL(t *testing.T) {
	cases := []struct {
		url, owner, repo string
	}{
		{"https://github.com/octocat/Hello-World.git", "octocat", "Hello-World"},
		{"git@github.com:octocat/Hello-World.git", "octocat", "Hello-World"},
		{"https://github.com/octocat/Hello-World", "octocat", "Hello-World"},
		{"ssh://git@github.com/octocat/Hello-World.git", "octocat", "Hello-World"},
		{"", "", ""},
	}
	for _, c := range cases {
		owner, repo := parseRemoteURL(c.url)
		if owner != c.owner || repo != c.repo {
			t.Errorf("parseRemoteURL(%q) = (%q,%q), want (%q,%q)", c.url, owner, repo, c.owner, c.repo)
		}
	}
}

func TestParseAuthor(t *testing.T) {
	name, email, ok := parseAuthor("张三 <zhang@example.com>")
	if !ok || name != "张三" || email != "zhang@example.com" {
		t.Errorf("parseAuthor = (%q,%q,%v), want (张三, zhang@example.com, true)", name, email, ok)
	}
	if _, _, ok := parseAuthor("no-brackets"); ok {
		t.Error("parseAuthor should fail without <>")
	}
}

func TestDefaultBranch(t *testing.T) {
	if DefaultBranch("") != "main" {
		t.Error("default branch should be main")
	}
	if DefaultBranch("trunk") != "trunk" {
		t.Error("custom branch should be respected")
	}
}
