package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

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

// setupRecipesBranch 创建菜谱独立分支、本地 worktree，并配置 main 忽略菜谱。
// 之后菜谱 CRUD 与构建都从 worktree（<root>/.recipes）读写。
func setupRecipesBranch(cmd *cobra.Command, projectRoot, branch string) error {
	// 1. 创建只含菜谱+配置的孤立分支（不打扰当前工作区与当前分支）
	paths := []string{".flavor-vault/recipes", ".flavor-vault/config.yaml"}
	if err := gitOrphanBranch(projectRoot, branch, "init: 初始化菜谱分支", paths); err != nil {
		return err
	}

	// 2. 建立 worktree
	wt := vault.RecipesWorktree(projectRoot)
	if _, err := os.Stat(wt); os.IsNotExist(err) {
		if out, err := gitRun(projectRoot, nil, "worktree", "add", wt, branch); err != nil {
			return fmt.Errorf("创建菜谱分支 worktree 失败: %w\n%s", err, out)
		}
	}

	// 3. main 忽略菜谱（菜谱只在独立分支维护，避免双份）
	if err := ensureGitignore(projectRoot, []string{".recipes/", ".flavor-vault/recipes/"}); err != nil {
		return err
	}

	// 4. 移除 main 工作区里的默认菜谱副本（真实数据在 worktree）
	if err := os.RemoveAll(vault.RecipesDir(projectRoot)); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✔ 已创建菜谱独立分支 %s（本地 worktree: %s）\n", branch, wt)
	fmt.Fprintf(cmd.OutOrStdout(), "  推送该分支：git push -u origin %s\n", branch)
	fmt.Fprintf(cmd.OutOrStdout(), "  之后维护菜谱：fv add / fv gh push --recipe <id>\n")
	return nil
}

// gitOrphanBranch 用临时索引创建"仅含指定路径"的孤立分支，不修改当前工作区。
func gitOrphanBranch(projectRoot, branch, message string, paths []string) error {
	tmpIndex := filepath.Join(os.TempDir(), fmt.Sprintf("fv-idx-%d-%s", time.Now().UnixNano(), branch))
	defer os.Remove(tmpIndex)
	env := append(os.Environ(), "GIT_INDEX_FILE="+tmpIndex)
	env = ensureAuthorEnv(env, projectRoot)

	run := func(args ...string) (string, error) { return gitRun(projectRoot, env, args...) }

	if out, err := run("read-tree", "--empty"); err != nil {
		return fmt.Errorf("准备索引失败: %w\n%s", err, out)
	}
	addArgs := append([]string{"add", "-f", "--"}, paths...)
	if out, err := run(addArgs...); err != nil {
		return fmt.Errorf("暂存菜谱失败: %w\n%s", err, out)
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
