package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	ghc "flavor-vault/internal/github"
	"flavor-vault/internal/models"
	"flavor-vault/internal/utils"
)

// recipeAPIClient 解析编辑目标（GitHub 仓库 + 分支）并构造 API 客户端。
// 编辑模式：add/edit/rm 经 GitHub API 直接操作数据源分支上的单文件，无需本地 clone/worktree。
// repo 来源：--repo / FV_REPO / config.source.repo 优先，其次 git remote（代码仓库）；
// 分支：--branch / FV_BRANCH / config.source.branch，默认 recipes；token 用 GITHUB_TOKEN 或 config.github.token。
func recipeAPIClient(cmd *cobra.Command) (*ghc.Client, string, string, error) {
	cfg, projectRoot, _, err := loadProjectConfig(cmd)
	if err != nil {
		return nil, "", "", err
	}
	cl, err := ghc.NewClientFromConfig(cfg)
	if err != nil {
		return nil, "", "", err
	}
	repo := strings.TrimSpace(cfg.Source.Repo)
	if repo == "" {
		owner, r, err := ghc.ResolveRepo(projectRoot)
		if err != nil {
			return nil, "", "", err
		}
		cl.Owner, cl.Repo = owner, r
	} else {
		owner, r, err := ghc.ParseRepoSpec(repo)
		if err != nil {
			return nil, "", "", err
		}
		cl.Owner, cl.Repo = owner, r
	}
	branch := strings.TrimSpace(cfg.Source.Branch)
	if branch == "" {
		branch = "recipes"
	}
	return cl, branch, projectRoot, nil
}

// apiRecipePath 数据源分支中的菜谱文件路径
func apiRecipePath(id string) string {
	return "recipes/" + id + ".json"
}

// apiListRecipes 列出数据源分支上的全部菜谱 ID
func apiListRecipes(ctx context.Context, cl *ghc.Client, branch string) ([]string, error) {
	names, err := cl.ListDir(ctx, branch, "recipes")
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(names))
	for _, n := range names {
		if strings.HasSuffix(n, ".json") {
			ids = append(ids, strings.TrimSuffix(n, ".json"))
		}
	}
	return ids, nil
}

// apiLoadRecipe 读取数据源分支上的单个菜谱
func apiLoadRecipe(ctx context.Context, cl *ghc.Client, branch, id string) (*models.Recipe, error) {
	raw, _, err := cl.GetFile(ctx, branch, apiRecipePath(id))
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("菜谱 %q 不存在", id)
	}
	var r models.Recipe
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("菜谱 %s 解析失败: %w", id, err)
	}
	return &r, nil
}

// apiSaveRecipe 提交/更新单个菜谱（含本地图片资源）到数据源分支，快进守卫防冲突。
// assetBase/assetDir 为本地暂存的图片目录（fv add 步骤配图已复制到那里）。
func apiSaveRecipe(ctx context.Context, cl *ghc.Client, branch string, r *models.Recipe, assetBase, assetDir, projectRoot, message string) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	files := map[string][]byte{apiRecipePath(r.ID): data}
	assetCount := 0
	for _, ref := range r.AssetRefs() {
		if utils.IsRemoteURL(ref) {
			continue
		}
		src := filepath.Join(assetDir, filepath.FromSlash(ref))
		b, err := os.ReadFile(src)
		if err != nil {
			continue // 资源缺失（如远程 URL 引用）跳过
		}
		files[filepath.ToSlash(filepath.Join(assetBase, ref))] = b
		assetCount++
	}

	author, err := ghc.ResolveAuthor(projectRoot, "")
	if err != nil {
		return err
	}
	res, err := cl.FastForwardPush(ctx, branch, message, files, author)
	if err != nil {
		return err
	}
	fmt.Printf("✔ 已提交 %s/%s@%s（commit %s，附 %d 个资源）\n",
		cl.Owner, cl.Repo, res.Branch, res.CommitSHA[:min(8, len(res.CommitSHA))], assetCount)
	return nil
}

// apiDeleteRecipe 删除数据源分支上的单个菜谱
func apiDeleteRecipe(ctx context.Context, cl *ghc.Client, branch, id, projectRoot, message string) error {
	_, sha, err := cl.GetFile(ctx, branch, apiRecipePath(id))
	if err != nil {
		return err
	}
	if sha == "" {
		return fmt.Errorf("菜谱 %q 不存在", id)
	}
	author, err := ghc.ResolveAuthor(projectRoot, "")
	if err != nil {
		return err
	}
	if err := cl.DeleteFile(ctx, branch, apiRecipePath(id), sha, message, author); err != nil {
		return err
	}
	return nil
}
