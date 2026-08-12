// build 是独立的静态站点数据构建器（ETL），供 CI（GitHub Actions）调用。
// fv CLI 制品不包含 build 命令；构建逻辑只在 CI 中完成。
//
// 用法：
//
//	go build -o build ./cmd/build
//	./build --force --output ./dist --asset-dir .flavor-vault/assets \
//	  --ai-snapshot --endpoint https://fv.sdniu.top/data
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	ghc "flavor-vault/internal/github"
	"flavor-vault/internal/models"
	"flavor-vault/internal/pipeline"
	"flavor-vault/internal/plugins"
	"flavor-vault/internal/store"
	"flavor-vault/internal/vault"
)

func main() {
	var (
		force      = flag.Bool("force", false, "强制重建（忽略缓存）")
		output     = flag.String("output", "", "覆盖输出目录（默认 dist）")
		assetDir   = flag.String("asset-dir", "", "图片资源目录（默认 .flavor-vault/assets）")
		aiSnapshot = flag.Bool("ai-snapshot", true, "是否生成 AI 快照 ai-corpus.json")
		endpoint   = flag.String("endpoint", "", "注入默认数据 endpoint 到 meta.json（也可用 FV_ENDPOINT）")
		config     = flag.String("config", "", "配置文件路径（默认 .flavor-vault/config.yaml）")
		sync       = flag.Bool("sync", false, "先从数据分支（recipes）拉取并映射 recipes/assets 到本地后再构建")
	)
	flag.Parse()
	start := time.Now()

	cfg, projectRoot, configPath, err := loadConfig(*config, *endpoint)
	if err != nil {
		fatal(err)
	}
	if *output != "" {
		cfg.OutputDir = *output
	}
	if *assetDir != "" {
		cfg.AssetDir = *assetDir
	}
	if !*aiSnapshot {
		cfg.AISnapshot = false
	}
	outDir := vault.ResolveOutputDir(projectRoot, cfg)

	// 可选：先从数据分支拉取数据（复刻 CI 合并），保证基于最新数据源码而非本地残留
	if *sync {
		if err := syncData(cfg, projectRoot); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ 数据同步失败，改用本地数据继续: %v\n", err)
		}
	}

	// 加载菜谱（格式错误文件跳过并警告，validator 强制校验）
	res, err := store.LoadAllMulti(
		[]string{vault.ResolveRecipesDir(projectRoot, cfg)},
		store.LoadOptions{SkipInvalid: true},
	)
	if err != nil {
		fatal(err)
	}
	for _, w := range res.Warnings {
		fmt.Fprintln(os.Stderr, "⚠", w)
	}

	ctx := pipeline.NewBuildContext(res.Recipes, cfg, outDir, vault.CacheRoot(projectRoot), configPath, *force)
	ctx.AssetDir = vault.ResolveAssetDir(projectRoot, cfg)
	ctx.Options["endpoint"] = *endpoint

	// 注册插件（按依赖顺序）
	scheduler := pipeline.NewScheduler(os.Stderr)
	scheduler.AddPlugin(&plugins.Validator{Logger: func(format string, a ...interface{}) {
		fmt.Fprintf(os.Stderr, format, a...)
	}})
	scheduler.AddPlugin(&plugins.FacetIndexer{})
	scheduler.AddPlugin(&plugins.TagIndexer{})
	scheduler.AddPlugin(&plugins.DetailSplitter{})
	scheduler.AddPlugin(&plugins.AssetCollector{})
	scheduler.AddPlugin(&plugins.StatsCollector{})
	scheduler.AddPlugin(&plugins.AIExporter{})
	scheduler.AddPlugin(&plugins.SearchIndexer{})

	if err := scheduler.Run(ctx); err != nil {
		fatal(err)
	}

	fmt.Fprintf(os.Stdout, "\n✔ 构建完成: %d 道菜谱 -> %s（耗时 %s）\n",
		len(res.Recipes), outDir, time.Since(start).Round(time.Millisecond))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "✗", err)
	os.Exit(1)
}

// loadConfig 解析构建配置：--config 优先，其次项目 .flavor-vault/config.yaml，再用户主目录；均无则用默认配置。
// --endpoint / FV_ENDPOINT、FV_REPO、FV_BRANCH 环境变量覆盖。
func loadConfig(configFlag, endpointFlag string) (*models.Config, string, string, error) {
	projectRoot, configPath, found := "", "", false
	if strings.TrimSpace(configFlag) != "" {
		root, cp, err := vault.ResolveWithConfig(configFlag)
		if err != nil {
			return nil, "", "", err
		}
		projectRoot, configPath, found = root, cp, true
	} else if root, err := vault.FindRoot(); err == nil {
		projectRoot, configPath, found = root, vault.ConfigPath(root), true
	} else if _, err := os.Stat(vault.HomeConfigPath()); err == nil {
		projectRoot, configPath, found = vault.HomeRoot(), vault.HomeConfigPath(), true
	} else {
		projectRoot, _ = os.Getwd()
	}

	var cfg *models.Config
	if found {
		c, err := vault.LoadConfigAt(configPath)
		if err != nil {
			return nil, "", "", err
		}
		cfg = c
	} else {
		cfg = models.DefaultConfig()
	}

	if strings.TrimSpace(endpointFlag) != "" {
		cfg.Endpoint = strings.TrimSpace(endpointFlag)
	} else if e := os.Getenv("FV_ENDPOINT"); strings.TrimSpace(e) != "" {
		cfg.Endpoint = strings.TrimSpace(e)
	}
	if e := os.Getenv("FV_REPO"); strings.TrimSpace(e) != "" {
		cfg.GitHub.Repo = strings.TrimSpace(e)
	}
	if e := os.Getenv("FV_BRANCH"); strings.TrimSpace(e) != "" {
		cfg.GitHub.Branch = strings.TrimSpace(e)
	}
	return cfg, projectRoot, configPath, nil
}

// syncData 按 gh-pages CI 的合并方式，从数据分支（recipes）拉取 recipes 与 assets，
// 映射到本地 .flavor-vault/recipes 与 .flavor-vault/assets（先清空避免残留）。
// 数据仓库来源：config github.repo 优先，其次 git remote origin；分支默认 recipes。
func syncData(cfg *models.Config, projectRoot string) error {
	owner, repo := "", ""
	if strings.TrimSpace(cfg.GitHub.Repo) != "" {
		o, r, err := ghc.ParseRepoSpec(cfg.GitHub.Repo)
		if err != nil {
			return err
		}
		owner, repo = o, r
	} else {
		o, r, err := ghc.ResolveRepo(projectRoot)
		if err != nil {
			return err
		}
		owner, repo = o, r
	}
	branch := strings.TrimSpace(cfg.GitHub.Branch)
	if branch == "" {
		branch = "recipes"
	}

	var cl *ghc.Client
	if os.Getenv("GITHUB_TOKEN") != "" || strings.TrimSpace(cfg.GitHub.Token) != "" {
		c, err := ghc.NewClientFromConfig(cfg)
		if err != nil {
			return err
		}
		c.Owner, c.Repo = owner, repo
		cl = c
	} else {
		cl = ghc.NewPublicClient(owner, repo)
	}

	files, err := cl.FetchTree(context.Background(), branch)
	if err != nil {
		return fmt.Errorf("拉取数据分支 %s 失败: %w", branch, err)
	}

	// 与 CI 相同的映射：recipes/ 与 .flavor-vault/recipes/ → .flavor-vault/recipes；.flavor-vault/assets/ 与 assets/ → .flavor-vault/assets
	recipesDst := filepath.Join(projectRoot, vault.DirName, vault.RecipesDirName)
	assetsDst := filepath.Join(projectRoot, vault.DirName, "assets")
	for _, d := range []string{recipesDst, assetsDst} {
		if err := os.RemoveAll(d); err != nil {
			return err
		}
	}

	count := 0
	for path, content := range files {
		var dst string
		switch {
		case strings.HasPrefix(path, "recipes/"):
			dst = filepath.Join(recipesDst, filepath.FromSlash(strings.TrimPrefix(path, "recipes/")))
		case strings.HasPrefix(path, vault.DirName+"/"+vault.RecipesDirName+"/"):
			dst = filepath.Join(recipesDst, filepath.FromSlash(strings.TrimPrefix(path, vault.DirName+"/"+vault.RecipesDirName+"/")))
		case strings.HasPrefix(path, vault.DirName+"/assets/"):
			dst = filepath.Join(assetsDst, filepath.FromSlash(strings.TrimPrefix(path, vault.DirName+"/assets/")))
		case strings.HasPrefix(path, "assets/"):
			dst = filepath.Join(assetsDst, filepath.FromSlash(strings.TrimPrefix(path, "assets/")))
		default:
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, content, 0o644); err != nil {
			return err
		}
		count++
	}
	fmt.Fprintf(os.Stdout, "✔ 已从 %s/%s@%s 同步 %d 个数据文件（recipes+assets）\n", owner, repo, branch, count)
	return nil
}
