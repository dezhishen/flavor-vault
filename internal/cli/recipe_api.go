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
	"flavor-vault/internal/vault"
)

// recipeAPIClient 解析编辑目标（GitHub 仓库 + 分支）并构造 API 客户端。
// 编辑模式：add/edit/rm 经 GitHub API 直接操作数据源分支上的单文件，无需本地 clone/worktree。
// repo 来源：--repo / FV_REPO / config.github.repo 优先，其次 git remote（代码仓库）；
// 分支：--branch / FV_BRANCH / config.github.branch，默认 recipes；token 用 GITHUB_TOKEN 或 config.github.token。
func recipeAPIClient(cmd *cobra.Command) (*ghc.Client, string, string, error) {
	cfg, projectRoot, _, err := loadProjectConfig(cmd)
	if err != nil {
		return nil, "", "", err
	}
	cl, err := ghc.NewClientFromConfig(cfg)
	if err != nil {
		return nil, "", "", err
	}
	repo := strings.TrimSpace(cfg.GitHub.Repo)
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
	branch := strings.TrimSpace(cfg.GitHub.Branch)
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
func apiSaveRecipe(ctx context.Context, cl *ghc.Client, branch string, r *models.Recipe, assetBase, assetDir string, cfg *models.Config, cfgPath, projectRoot, message string) error {
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

	author, err := ensureAuthor(ctx, cl, cfg, cfgPath, projectRoot)
	if err != nil {
		return err
	}
	res, err := cl.FastForwardPush(ctx, branch, message, files, author)
	if err != nil {
		return err
	}
	fmt.Printf("✔ 已提交 %s/%s@%s（commit %s，作者 %s，附 %d 个资源）\n",
		cl.Owner, cl.Repo, res.Branch, res.CommitSHA[:min(8, len(res.CommitSHA))], author.Name, assetCount)
	return nil
}

// apiDeleteRecipe 删除数据源分支上的单个菜谱
func apiDeleteRecipe(ctx context.Context, cl *ghc.Client, branch, id string, cfg *models.Config, cfgPath, projectRoot, message string) error {
	_, sha, err := cl.GetFile(ctx, branch, apiRecipePath(id))
	if err != nil {
		return err
	}
	if sha == "" {
		return fmt.Errorf("菜谱 %q 不存在", id)
	}
	author, err := ensureAuthor(ctx, cl, cfg, cfgPath, projectRoot)
	if err != nil {
		return err
	}
	if err := cl.DeleteFile(ctx, branch, apiRecipePath(id), sha, message, author); err != nil {
		return err
	}
	return nil
}

// ensureAuthor 确定提交作者（维护模式 token 必填）：
//  1. 配置 author{name,email}（显式覆盖）
//  2. 基于 token 自动获取（GET /user）并**写入配置**（author.name/author.email），供后续复用
//  3. 本地 git config 兜底
//
// 若仍无法确定（如 API 未返回姓名/邮箱），要求用户手动补充。
func ensureAuthor(ctx context.Context, cl *ghc.Client, cfg *models.Config, cfgPath, projectRoot string) (ghc.Author, error) {
	// 1. 配置作者覆盖
	if cfg != nil {
		name := strings.TrimSpace(cfg.Author.Name)
		email := strings.TrimSpace(cfg.Author.Email)
		if name != "" && email != "" {
			return ghc.Author{Name: name, Email: email}, nil
		}
	}
	// 2. 基于 token 自动获取并持久化到配置
	if cl != nil {
		if u, err := cl.CurrentUser(ctx); err == nil {
			name := u.GetName()
			if name == "" {
				name = u.GetLogin()
			}
			email := u.GetEmail()
			if email == "" && u.GetID() != 0 && u.GetLogin() != "" {
				// GitHub 不提供公开邮箱时，用 noreply 邮箱（GitHub 推荐格式）
				email = fmt.Sprintf("%d+%s@users.noreply.github.com", u.GetID(), u.GetLogin())
			}
			if name != "" && email != "" && cfg != nil {
				cfg.Author.Name = name
				cfg.Author.Email = email
				_ = persistAuthor(cfg, cfgPath, projectRoot)
				return ghc.Author{Name: name, Email: email}, nil
			}
		}
	}
	// 3. git config 兜底
	if a, err := ghc.ResolveAuthor(projectRoot, ""); err == nil {
		return a, nil
	}
	return ghc.Author{}, fmt.Errorf("无法确定提交作者：请运行 fv config set author.name <姓名> / fv config set author.email <邮箱>（维护模式需配置作者）")
}

// persistAuthor 将解析到的作者写入配置（未指定 cfgPath 时用默认 .flavor-vault/config.yaml）
func persistAuthor(cfg *models.Config, cfgPath, projectRoot string) error {
	path := strings.TrimSpace(cfgPath)
	if path == "" {
		path = vault.ConfigPath(projectRoot)
	}
	return vault.SaveConfigAt(path, cfg)
}
