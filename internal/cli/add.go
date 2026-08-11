package cli

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"flavor-vault/internal/data"
	"flavor-vault/internal/models"
	"flavor-vault/internal/plugins"
	"flavor-vault/internal/utils"
)

func newAddCmd() *cobra.Command {
	var jsonInput string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "创建新菜谱（交互式，或 --json 直接输入；可用 --action-id 缓存草稿）",
		Args:  cobra.NoArgs,
		Example: `  fv add                                          # 交互式
  fv add --json '{"name":"红烧肉","tags":["热菜"]}'  # 直接提供 JSON
  fv add --json @recipe.json                       # 从文件读取
  fv add --action-id abc123 --json @recipe.json    # 缓存草稿，失败可续写`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}

			st := actionStoreFor(cmd)

			// 1. 从缓存恢复草稿
			r := &models.Recipe{}
			if cached, ok := loadCachedRecipe(st, "add"); ok && cached != nil {
				r = cached
				fmt.Fprintf(cmd.ErrOrStderr(), "ℹ 已从缓存恢复草稿（action-id=%s）\n", st.ID)
			}

			// 2. --json 输入（覆盖草稿）
			if jsonInput != "" {
				if err := parseRecipeJSON(jsonInput, r); err != nil {
					return err
				}
			} else {
				// 3. 交互式收集（以草稿为默认值，支持续写）
				reader := bufio.NewReader(os.Stdin)
				r, err = promptAddRecipe(reader, cfg, projectRoot, r)
				if err != nil {
					return err
				}
			}

			// 4. 补齐元数据
			now := time.Now()
			if r.CreatedAt.IsZero() {
				r.CreatedAt = now
			}
			r.UpdatedAt = now
			if r.ID == "" {
				r.ID = generateID(r.Name)
			}

			// 5. 校验；有误则缓存草稿供重试
			if err := plugins.ValidateRecipe(r, cfg); err != nil {
				return failAndCache(cmd, st, "add", r.ID, r, err)
			}

			// 6. 经 GitHub API 提交到数据源分支（单文件 + 本地图片资源，无需本地 clone）
			ctx := context.Background()
			cl, branch, projectRoot, err := recipeAPIClient(cmd)
			if err != nil {
				return failAndCache(cmd, st, "add", r.ID, r, err)
			}
			exists, err := cl.FileExists(ctx, branch, apiRecipePath(r.ID))
			if err != nil {
				return failAndCache(cmd, st, "add", r.ID, r, err)
			}
			if exists {
				return failAndCache(cmd, st, "add", r.ID, r, fmt.Errorf("菜谱 %q 已存在", r.ID))
			}
			assetBase := cfg.AssetDir
			if assetBase == "" {
				assetBase = ".flavor-vault/assets"
			}
			if err := apiSaveRecipe(ctx, cl, branch, r, assetBase, assetDirFor(cfg, projectRoot), projectRoot,
				fmt.Sprintf("add: %s", r.Name)); err != nil {
				return failAndCache(cmd, st, "add", r.ID, r, err)
			}
			completeAction(cmd, st)
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已创建并提交菜谱 %s (%s)\n", r.ID, r.Name)
			return nil
		},
	}
	cmd.Flags().StringVar(&jsonInput, "json", "", "直接以 JSON 提供菜谱（支持 @文件路径），跳过交互")
	return cmd
}

// promptAddRecipe 交互式收集菜谱字段，使用 draft 中的值作为默认值
func promptAddRecipe(reader *bufio.Reader, cfg *models.Config, projectRoot string, draft *models.Recipe) (*models.Recipe, error) {
	r := &models.Recipe{}
	if draft != nil {
		*r = *draft
	}

	name, err := prompt(reader, "菜名", r.Name)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("菜名不能为空")
	}
	r.Name = name

	desc, _ := prompt(reader, "简介", r.Description)
	r.Description = desc

	// 可选标签/厨具来自已有数据（本地 dist/data 或远程 endpoint 的 filters.json），仅作提示
	availTags, availKw := availableFacets(cfg, projectRoot)

	fmt.Println("\n可用标签（来自已有数据）:", strings.Join(availTags, ", "))
	tags, err := promptCSV(reader, "标签", "")
	if err != nil {
		return nil, err
	}
	if len(tags) > 0 {
		r.Tags = tags
	}

	fmt.Println("\n可用厨具（来自已有数据）:", strings.Join(availKw, ", "))
	kw, err := promptCSV(reader, "厨具", "")
	if err != nil {
		return nil, err
	}
	if len(kw) > 0 {
		r.Kitchenware = kw
	}

	// 主要食材
	if len(r.Ingredients.Main) == 0 {
		for {
			more, _ := promptBool(reader, "添加主要食材?", len(r.Ingredients.Main) == 0)
			if !more {
				break
			}
			ing, err := promptIngredient(reader, "主要")
			if err != nil {
				return nil, err
			}
			r.Ingredients.Main = append(r.Ingredients.Main, ing)
		}
	} else {
		fmt.Fprintln(os.Stderr, "ℹ 主要食材已存在（来自草稿），如需修改请用 --json 或 fv edit")
	}

	// 配菜/辅料
	moreSide, _ := promptBool(reader, "添加配菜/辅料?", len(r.Ingredients.Side) > 0)
	for moreSide {
		ing, err := promptIngredient(reader, "辅料")
		if err != nil {
			return nil, err
		}
		r.Ingredients.Side = append(r.Ingredients.Side, ing)
		moreSide, _ = promptBool(reader, "继续添加辅料?", false)
	}

	// 步骤（自然语言：第一步/第二步…，每步之间可插图片）
	if len(r.Steps) == 0 {
		for {
			n := len(r.Steps) + 1
			desc, err := prompt(reader, fmt.Sprintf("第 %d 步做什么？（直接回车结束步骤）", n), "")
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(desc) == "" {
				break
			}
			img, _ := prompt(reader, fmt.Sprintf("第 %d 步配图（本地路径将复制到 assets / 图片URL，回车跳过）", n), "")
			imgRef := ""
			if strings.TrimSpace(img) != "" {
				ref, err := resolveStepImage(projectRoot, cfg, r.Name, n, strings.TrimSpace(img))
				if err != nil {
					fmt.Fprintf(os.Stderr, "⚠ 配图失败（已跳过）: %v\n", err)
				} else {
					imgRef = ref
				}
			}
			r.Steps = append(r.Steps, models.Step{Order: n, Description: desc, ImageRef: imgRef})
		}
	} else {
		fmt.Fprintln(os.Stderr, "ℹ 步骤已存在（来自草稿），如需修改请用 --json 或 fv edit")
	}

	prepTime, _ := promptInt(reader, "准备时间(分钟)", orDefault(r.Stats.PrepTime, 10))
	cookTime, _ := promptInt(reader, "烹饪时间(分钟)", orDefault(r.Stats.CookTime, 15))
	difficulty, _ := promptInt(reader, "难度(1-5)", defaultDifficulty(r.Stats.Difficulty))
	r.Stats.PrepTime = prepTime
	r.Stats.CookTime = cookTime
	r.Stats.Difficulty = difficulty

	id, _ := prompt(reader, "ID（回车自动生成）", r.ID)
	if id != "" {
		r.ID = id
	}
	return r, nil
}

func defaultDifficulty(d int) int {
	if d >= 1 && d <= 5 {
		return d
	}
	return 2
}

// resolveStepImage 处理步骤配图：本地文件复制到 assets/images/ 并返回引用，外部 URL 原样返回。
// 命名规范：<菜谱名>-<步骤>-<序号>，如 红烧肉-2-1.png
func resolveStepImage(projectRoot string, cfg *models.Config, recipeName string, step int, src string) (string, error) {
	if utils.IsRemoteURL(src) {
		return src, nil
	}
	info, err := os.Stat(src)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("图片不存在: %s", src)
	}
	ext := filepath.Ext(src)
	if ext == "" {
		ext = ".img"
	}
	base := sanitizeName(recipeName)
	if base == "" {
		base = "recipe"
	}
	name := fmt.Sprintf("%s-%d-1%s", base, step, ext)
	dst := filepath.Join(assetDirFor(cfg, projectRoot), "images", name)
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return "", err
	}
	return "images/" + name, nil
}

// sanitizeName 清洗文件名字段：仅去除文件系统/URL 非法字符与空白，保留中文
func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`[\\/:*?"<>|\s]+`)
	return strings.Trim(re.ReplaceAllString(s, "-"), "-")
}

// orDefault 值非零时返回 v，否则返回默认值（用于交互提示的默认值）
func orDefault(v, def int) int {
	if v != 0 {
		return v
	}
	return def
}

func promptIngredient(reader *bufio.Reader, kind string) (models.Ingredient, error) {
	name, err := prompt(reader, kind+"食材名称", "")
	if err != nil {
		return models.Ingredient{}, err
	}
	if name == "" {
		return models.Ingredient{}, fmt.Errorf("食材名称不能为空")
	}
	amount, _ := prompt(reader, "用量（如 500g/适量）", "适量")
	return models.Ingredient{Name: name, Amount: amount}, nil
}

var latinSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

// generateID 生成 ID：拉丁字符转 slug；否则用随机后缀
func generateID(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	if isLatin(slug) {
		slug = latinSlugRe.ReplaceAllString(slug, "-")
		slug = strings.Trim(slug, "-")
		if slug != "" {
			return slug
		}
	}
	return "recipe-" + randomHex(6)
}

func isLatin(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return s != ""
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n*2]
}

// availableFacets 从 facet 索引读取已有标签与厨具（本地 dist/data 或远程 endpoint 的
// filters.json），用于交互式添加菜谱时的提示。数据不存在时返回空，不报错。
func availableFacets(cfg *models.Config, projectRoot string) (tags, kitchenware []string) {
	locator, remote := data.Locator(cfg, projectRoot, "filters.json")
	raw, err := data.ReadJSON(locator, remote)
	if err != nil {
		return nil, nil
	}
	var idx struct {
		Tags        map[string][]string `json:"tags"`
		Kitchenware map[string][]string `json:"kitchenware"`
	}
	if err := json.Unmarshal(raw, &idx); err != nil {
		return nil, nil
	}
	return sortedMapKeys(idx.Tags), sortedMapKeys(idx.Kitchenware)
}

func sortedMapKeys(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
