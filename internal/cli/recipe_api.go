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
	cfg, projectRoot, cfgPath, err := loadProjectConfig(cmd)
	if err != nil {
		return nil, "", "", err
	}

	// 按需补全编辑配置（降低门槛：只有真正编辑菜谱时才询问更多信息）。
	// 交互式下缺 repo/token 时引导输入并写入配置，避免直接报错；非交互仍走参数/环境变量。
	if isInteractive() {
		changed := false
		reader := newLineReader()
		if strings.TrimSpace(cfg.GitHub.Repo) == "" && !cmd.Flags().Changed("repo") {
			def := ""
			if owner, r, err := ghc.ResolveRepo(projectRoot); err == nil {
				def = owner + "/" + r
			}
			if v, err := prompt(reader, "GitHub 数据仓库（owner/repo，菜谱存放处）", def); err == nil {
				cfg.GitHub.Repo = strings.TrimSpace(v)
				changed = true
			}
		}
		if strings.TrimSpace(cfg.GitHub.Token) == "" && strings.TrimSpace(os.Getenv("GITHUB_TOKEN")) == "" && !cmd.Flags().Changed("token") {
			if v, err := prompt(reader, "GitHub Token（需该仓库写权限；也可用 GITHUB_TOKEN 环境变量）", ""); err == nil {
				cfg.GitHub.Token = strings.TrimSpace(v)
				changed = true
			}
		}
		if changed {
			_ = persistConfig(cfg, cfgPath, projectRoot)
		}
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
			// 分支已有资产（images/ 前缀，本地无原文件）保持原样不重复上传；其余本地资源缺失则报错
			if strings.HasPrefix(ref, "images/") {
				continue
			}
			return fmt.Errorf("本地资源缺失，无法提交: %s（请将图片复制到 %s 或以相对路径引用）", ref, assetDir)
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

// stageLocalAssets 把菜谱引用的本地图片复制到资源目录（assetDir）并就地更新引用，
// 供 add --json / edit / gh push --recipe 等路径复用（交互式 add 已复制过则原样保留）。
// 规则：
//   - 远程 URL（http/https/data:）原样保留；
//   - 引用规范化：去掉 assetBase 前缀（.flavor-vault/assets/）→ 相对引用；
//   - 已在资源目录内，或为分支已有资产（images/ 前缀，本地无文件）：原样保留；
//   - 其余按本地文件复制，同一菜谱的图片集中存放于 assets/images/<recipeID>/，
//     步骤图命名 <菜谱名>-<步骤>-<序号>，封面/过程图 <菜谱名>-cover/img-N。
//
// 返回暂存的图片数；任一本地图片缺失则返回错误。
func stageLocalAssets(cfg *models.Config, projectRoot string, r *models.Recipe) (int, error) {
	dir := assetDirFor(cfg, projectRoot)
	assetBase := ".flavor-vault/assets"
	if cfg != nil && strings.TrimSpace(cfg.AssetDir) != "" {
		assetBase = strings.TrimSpace(cfg.AssetDir)
	}
	base := sanitizeName(r.Name)
	if base == "" {
		base = "recipe"
	}
	// 图片按菜谱分组：<assets>/images/<recipeID>/（ID 缺失时用菜谱名）
	sub := strings.TrimSpace(r.ID)
	if sub == "" {
		sub = base
	}
	imagesDir := filepath.Join(dir, "images", sub)
	staged := 0

	stage := func(ref, hint string) (string, error) {
		ref = strings.TrimSpace(ref)
		if ref == "" || utils.IsRemoteURL(ref) {
			return ref, nil
		}
		// 规范化：去掉 assetBase 前缀（.flavor-vault/assets/）得到 asset 相对引用
		norm := strings.TrimPrefix(strings.TrimPrefix(ref, assetBase+"/"), "./"+assetBase+"/")
		// 1) 已在资源目录内（含已暂存/用户放置）→ 原样（规范化后）
		if !filepath.IsAbs(norm) {
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(norm))); err == nil {
				return norm, nil
			}
		}
		// 2) 分支已有资产（images/ 前缀，本地无原文件）→ 原样，不重复上传
		if strings.HasPrefix(norm, "images/") && !filepath.IsAbs(norm) {
			return norm, nil
		}
		// 3) 本地源文件：cwd 相对，或 assetBase 相对
		candidates := []string{ref}
		if !filepath.IsAbs(ref) {
			candidates = append(candidates, filepath.ToSlash(filepath.Join(assetBase, norm)))
		}
		var src string
		for _, cand := range candidates {
			if info, err := os.Stat(cand); err == nil && !info.IsDir() {
				src = cand
				break
			}
		}
		if src == "" {
			return "", fmt.Errorf("本地图片不存在: %s（请将图片放到 %s 或以相对路径引用）", ref, dir)
		}
		// 复制到 images/<recipeID>/ 并更新引用（命名 <菜谱名>-<hint>[-<序号>]）
		ext := filepath.Ext(src)
		if ext == "" {
			ext = ".img"
		}
		if err := os.MkdirAll(imagesDir, 0o755); err != nil {
			return "", err
		}
		name := fmt.Sprintf("%s-%s%s", base, hint, ext)
		dst := filepath.Join(imagesDir, name)
		for i := 2; ; i++ {
			if _, err := os.Stat(dst); os.IsNotExist(err) {
				break
			}
			name = fmt.Sprintf("%s-%s-%d%s", base, hint, i, ext)
			dst = filepath.Join(imagesDir, name)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return "", err
		}
		return "images/" + sub + "/" + name, nil
	}

	if v, err := stage(r.Media.Cover, "cover"); err != nil {
		return staged, err
	} else if v != r.Media.Cover {
		staged++
		r.Media.Cover = v
	}
	for i := range r.Media.Images {
		if v, err := stage(r.Media.Images[i], fmt.Sprintf("img-%d", i+1)); err != nil {
			return staged, err
		} else if v != r.Media.Images[i] {
			staged++
			r.Media.Images[i] = v
		}
	}
	for i := range r.Steps {
		hint := fmt.Sprintf("%d-1", r.Steps[i].Order)
		if v, err := stage(r.Steps[i].ImageRef, hint); err != nil {
			return staged, err
		} else if v != r.Steps[i].ImageRef {
			staged++
			r.Steps[i].ImageRef = v
		}
	}
	return staged, nil
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

// persistConfig 将配置写回（未指定 cfgPath 时用默认 .flavor-vault/config.yaml）
func persistConfig(cfg *models.Config, cfgPath, projectRoot string) error {
	path := strings.TrimSpace(cfgPath)
	if path == "" {
		path = vault.ConfigPath(projectRoot)
	}
	return vault.SaveConfigAt(path, cfg)
}

// persistAuthor 将解析到的作者写入配置（未指定 cfgPath 时用默认 .flavor-vault/config.yaml）
func persistAuthor(cfg *models.Config, cfgPath, projectRoot string) error {
	return persistConfig(cfg, cfgPath, projectRoot)
}
