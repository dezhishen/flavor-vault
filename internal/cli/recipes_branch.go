package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
	"flavor-vault/internal/vault"
)

// gitRun 在项目根执行 git 命令
func gitRun(projectRoot string, env []string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = projectRoot
	c.Env = env
	out, err := c.CombinedOutput()
	return string(out), err
}

// setupRecipesBranch 创建菜谱"数据仓库"分支（recipes）：
// 该分支自带 config.yaml + recipes + assets + .gitignore + README，
// 可被独立 fork / 私有化复用。本地以 worktree（<root>/.recipes）检出。
func setupRecipesBranch(cmd *cobra.Command, projectRoot, branch string, cfg *models.Config) error {
	// 1. 组装数据仓库内容到临时目录
	tmpDir, paths, err := buildDataRepo(projectRoot, cfg)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// 2. 从临时目录创建孤立分支（不打扰当前工作区）
	if err := gitOrphanBranchFromDir(projectRoot, tmpDir, branch, "init: 初始化菜谱数据仓库", paths); err != nil {
		return err
	}

	// 3. 建立本地 worktree（数据仓库检出）
	wt := vault.RecipesWorktree(projectRoot)
	if _, err := os.Stat(wt); os.IsNotExist(err) {
		if out, err := gitRun(projectRoot, nil, "worktree", "add", wt, branch); err != nil {
			return fmt.Errorf("创建菜谱数据仓库 worktree 失败: %w\n%s", err, out)
		}
	}

	// 4. main 忽略数据（recipes/assets 只存在于数据仓库）
	if err := ensureGitignore(projectRoot, []string{".recipes/", ".flavor-vault/recipes/", ".flavor-vault/assets/"}); err != nil {
		return err
	}

	// 5. 移除 main 工作区里的本地数据副本（真实数据在 worktree）
	_ = os.RemoveAll(filepath.Join(projectRoot, vault.DirName, vault.RecipesDirName))
	_ = os.RemoveAll(filepath.Join(projectRoot, filepath.FromSlash(defaultAssetBase(cfg))))

	fmt.Fprintf(cmd.OutOrStdout(), "✔ 已创建菜谱数据仓库分支 %s（本地 worktree: %s）\n", branch, wt)
	fmt.Fprintf(cmd.OutOrStdout(), "  推送该分支：git push -u origin %s\n", branch)
	fmt.Fprintf(cmd.OutOrStdout(), "  数据仓库内容：config.yaml + recipes/ + assets/ + README（可独立 fork/私有化）\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  之后维护菜谱：fv add / fv gh push --recipe <id>\n")
	return nil
}

// buildDataRepo 把 config.yaml、recipes、assets 组装成可独立复用的数据仓库临时目录
func buildDataRepo(projectRoot string, cfg *models.Config) (string, []string, error) {
	tmp := filepath.Join(os.TempDir(), fmt.Sprintf("fv-data-%d", time.Now().UnixNano()))
	paths := []string{}

	// config.yaml
	cfgSrc := vault.ConfigPath(projectRoot)
	if data, err := os.ReadFile(cfgSrc); err == nil {
		dst := filepath.Join(tmp, vault.DirName, vault.ConfigName)
		if werr := writeFileAll(dst, data); werr != nil {
			return "", nil, werr
		}
		paths = append(paths, filepath.ToSlash(filepath.Join(vault.DirName, vault.ConfigName)))
	}

	// recipes
	rcp := vault.RecipesDir(projectRoot)
	if n, err := copyTree(rcp, filepath.Join(tmp, vault.DirName, vault.RecipesDirName)); err == nil && n > 0 {
		paths = append(paths, filepath.ToSlash(filepath.Join(vault.DirName, vault.RecipesDirName)))
	}

	// assets（图片等资源）
	ast := vault.ResolveAssetDir(projectRoot, cfg)
	assetBase := defaultAssetBase(cfg)
	if n, err := copyTree(ast, filepath.Join(tmp, filepath.FromSlash(assetBase))); err == nil && n > 0 {
		paths = append(paths, filepath.ToSlash(assetBase))
	}

	// 数据仓库自己的 .gitignore（忽略缓存与锁）
	if err := writeFileAll(filepath.Join(tmp, ".gitignore"),
		[]byte(".flavor-vault/cache/\n.flavor-vault/push.lock\n")); err != nil {
		return "", nil, err
	}
	paths = append(paths, ".gitignore")

	// 数据仓库 README（方便 fork / 私有化复用）
	if err := writeFileAll(filepath.Join(tmp, "README.md"), []byte(dataRepoReadme(cfg))); err != nil {
		return "", nil, err
	}
	paths = append(paths, "README.md")

	return tmp, paths, nil
}

func defaultAssetBase(cfg *models.Config) string {
	if cfg != nil && strings.TrimSpace(cfg.AssetDir) != "" {
		return strings.TrimSpace(cfg.AssetDir)
	}
	return ".flavor-vault/assets"
}

// dataRepoReadme 生成数据仓库的自述（面向 fork/私有化）
func dataRepoReadme(cfg *models.Config) string {
	return fmt.Sprintf(`# Flavor Vault · 菜谱数据仓库

本分支是独立的**菜谱数据仓库**，仅包含数据，不含程序代码：

- .flavor-vault/config.yaml — 数据侧配置（标签白名单、资源目录等）
- .flavor-vault/recipes/ — 菜谱源文件（每道菜一个 JSON）
- .flavor-vault/assets/ — 图片等资源

## 使用方式

- **独立使用**：克隆本分支即为一个完整数据源，配合 Flavor Vault CLI 读取：
  `+"`"+`git clone -b %s <repo-url> data && cd data && fv list`+"`"+`
- **fork / 私有化**：直接 fork 本分支，修改 config.yaml 与 recipes 即可，程序端零改动。
- **代码侧**：程序代码在默认分支（main），本分支只维护数据。

## 维护

    fv add                          # 新增菜谱（写入本仓库 recipes/）
    fv gh push --recipe <id>        # 把单个菜谱连同图片提交到本分支
`, defaultAssetBase(cfg))
}

// gitOrphanBranchFromDir 用临时索引 + --work-tree 从 srcDir 创建孤立分支
func gitOrphanBranchFromDir(projectRoot, srcDir, branch, message string, paths []string) error {
	tmpIndex := filepath.Join(os.TempDir(), fmt.Sprintf("fv-idx-%d-%s", time.Now().UnixNano(), branch))
	defer os.Remove(tmpIndex)
	env := append(os.Environ(), "GIT_INDEX_FILE="+tmpIndex)
	env = ensureAuthorEnv(env, projectRoot)

	run := func(args ...string) (string, error) { return gitRun(projectRoot, env, args...) }

	if out, err := run("read-tree", "--empty"); err != nil {
		return fmt.Errorf("准备索引失败: %w\n%s", err, out)
	}
	// 从 srcDir 暂存（--work-tree），路径相对 srcDir
	addArgs := append([]string{"--work-tree=" + srcDir, "add", "-f", "--"}, paths...)
	if out, err := run(addArgs...); err != nil {
		return fmt.Errorf("暂存数据仓库失败: %w\n%s", err, out)
	}
	treeOut, err := run("write-tree")
	if err != nil {
		return fmt.Errorf("写树失败: %w\n%s", err, treeOut)
	}
	tree := strings.TrimSpace(treeOut)

	commitOut, err := run("commit-tree", tree, "-m", message)
	if err != nil {
		return fmt.Errorf("创建提交失败: %w\n%s", err, commitOut)
	}
	commit := strings.TrimSpace(commitOut)

	if out, err := run("update-ref", "refs/heads/"+branch, commit); err != nil {
		return fmt.Errorf("更新分支 %s 失败: %w\n%s", branch, err, out)
	}
	return nil
}

// ensureAuthorEnv 保证提交身份存在（缺省用 Flavor Vault 兜底）
func ensureAuthorEnv(env []string, projectRoot string) []string {
	get := func(k string) string {
		out, err := gitRun(projectRoot, nil, "config", "--get", k)
		if err != nil {
			return ""
		}
		return strings.TrimSpace(out)
	}
	name, email := get("user.name"), get("user.email")
	if name == "" {
		name = "Flavor Vault"
	}
	if email == "" {
		email = "fv@flavor-vault.local"
	}
	return append(env,
		"GIT_AUTHOR_NAME="+name, "GIT_AUTHOR_EMAIL="+email,
		"GIT_COMMITTER_NAME="+name, "GIT_COMMITTER_EMAIL="+email)
}

// ensureGitignore 向项目根 .gitignore 追加缺失条目
func ensureGitignore(projectRoot string, entries []string) error {
	path := filepath.Join(projectRoot, ".gitignore")
	content := ""
	if data, err := os.ReadFile(path); err == nil {
		content = string(data)
	}
	changed := false
	for _, e := range entries {
		if !strings.Contains(content, e) {
			content += e + "\n"
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// writeFileAll 写文件（自动建目录）
func writeFileAll(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// copyTree 递归复制目录，返回复制文件数
func copyTree(src, dst string) (int, error) {
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return 0, nil
	}
	count := 0
	err = filepath.Walk(src, func(p string, fi os.FileInfo, e error) error {
		if e != nil {
			return nil
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		if werr := writeFileAll(target, data); werr == nil {
			count++
		}
		return nil
	})
	return count, err
}

