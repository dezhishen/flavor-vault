package vault

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"flavor-vault/internal/models"
)

const (
	// DirName 本地数据目录名
	DirName = ".flavor-vault"
	// ConfigName 配置文件
	ConfigName = "config.yaml"
	// RecipesDirName 菜谱目录
	RecipesDirName = "recipes"
	// CacheDirName 缓存目录
	CacheDirName = "cache"
)

// FindRoot 从当前目录向上查找包含 .flavor-vault 的根目录
func FindRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, DirName)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("未找到 %s 目录，请先运行 fv init", DirName)
}

// ResolveWithConfig 根据显式配置文件路径解析项目根。
// 标准布局 <root>/.flavor-vault/config.yaml → root 为 .flavor-vault 的上级目录；
// 其他自定义路径 → root 为配置文件所在目录。
func ResolveWithConfig(configPath string) (projectRoot, resolvedConfigPath string, err error) {
	abs, err := filepath.Abs(configPath)
	if err != nil {
		return "", "", err
	}
	if filepath.Base(abs) == ConfigName && filepath.Base(filepath.Dir(abs)) == DirName {
		return filepath.Dir(filepath.Dir(abs)), abs, nil
	}
	return filepath.Dir(abs), abs, nil
}

// ResolveContext 根据 --config 标志解析项目上下文（root + configPath）。
// configFlag 为空时自动查找 .flavor-vault。
func ResolveContext(configFlag string) (projectRoot, configPath string, err error) {
	flag := strings.TrimSpace(configFlag)
	if flag == "" {
		root, err := FindRoot()
		if err != nil {
			return "", "", err
		}
		return root, ConfigPath(root), nil
	}
	return ResolveWithConfig(flag)
}

// LoadConfigAt 从显式路径加载配置；若文件不存在则使用默认配置
func LoadConfigAt(configPath string) (*models.Config, error) {
	cfg := models.DefaultConfig()
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytesReader(data)); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", configPath, err)
	}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", configPath, err)
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "./dist"
	}
	return cfg, nil
}

// LoadConfigOptional 加载配置（配置可选，不报错）：
// 优先 --config 指定路径，其次查找 .flavor-vault/config.yaml，都没有则返回默认配置。
// 返回 (配置, 项目根, 配置文件路径)。
func LoadConfigOptional(configFlag string) (*models.Config, string, string) {
	if flag := strings.TrimSpace(configFlag); flag != "" {
		if root, cp, err := ResolveWithConfig(flag); err == nil {
			if cfg, err := LoadConfigAt(cp); err == nil {
				return cfg, root, cp
			}
		}
	}
	if root, err := FindRoot(); err == nil {
		cp := ConfigPath(root)
		if cfg, err := LoadConfigAt(cp); err == nil {
			return cfg, root, cp
		}
	}
	dir, _ := os.Getwd()
	return models.DefaultConfig(), dir, ""
}

// LoadConfig 加载配置；若文件不存在则使用默认配置
func LoadConfig(projectRoot string) (*models.Config, string, error) {
	path := ConfigPath(projectRoot)
	cfg, err := LoadConfigAt(path)
	return cfg, path, err
}

// VaultRoot 返回 .flavor-vault 目录路径
func VaultRoot(projectRoot string) string {
	return filepath.Join(projectRoot, DirName)
}

// RecipesDir 返回菜谱目录
func RecipesDir(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, RecipesDirName)
}

// CacheRoot 返回缓存目录
func CacheRoot(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, CacheDirName)
}

// SourceDir 返回唯一菜谱数据源的本地检出目录（.flavor-vault/source）
func SourceDir(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, "source")
}

// SourceRecipesDir 返回数据源检出中的菜谱目录。
// 兼容两种布局：数据仓库式 .flavor-vault/recipes 或根目录 recipes/。
func SourceRecipesDir(projectRoot string) string {
	base := SourceDir(projectRoot)
	standard := filepath.Join(base, DirName, RecipesDirName)
	if info, err := os.Stat(standard); err == nil && info.IsDir() {
		return standard
	}
	root := filepath.Join(base, RecipesDirName)
	if info, err := os.Stat(root); err == nil && info.IsDir() {
		return root
	}
	return standard // 不存在时返回标准路径（加载时会跳过）
}

// SourceAssetDir 返回数据源检出中的资源目录（默认 .flavor-vault/assets）
func SourceAssetDir(projectRoot string, cfg *models.Config) string {
	base := ".flavor-vault/assets"
	if cfg != nil && strings.TrimSpace(cfg.AssetDir) != "" {
		base = cfg.AssetDir
	}
	return filepath.Join(SourceDir(projectRoot), filepath.FromSlash(base))
}

// RecipesWorktree 返回同仓库菜谱独立分支的本地 worktree 目录（<root>/.recipes）
func RecipesWorktree(projectRoot string) string {
	return filepath.Join(projectRoot, ".recipes")
}

// ResolveRecipesDir 解析菜谱源目录（维护者数据源检出优先）：
// 1) 同仓库独立分支 worktree（<root>/.recipes）
// 2) 独立仓库数据源检出（<root>/.flavor-vault/source）
// 3) 默认 <root>/.flavor-vault/recipes，其次根目录 <root>/recipes/（fv add 现用布局）
func ResolveRecipesDir(projectRoot string, cfg *models.Config) string {
	if cfg != nil && cfg.Maintainer() {
		wt := filepath.Join(RecipesWorktree(projectRoot), DirName, RecipesDirName)
		if info, err := os.Stat(wt); err == nil && info.IsDir() {
			return wt
		}
		src := SourceRecipesDir(projectRoot)
		if info, err := os.Stat(src); err == nil && info.IsDir() {
			return src
		}
	}
	if d := RecipesDir(projectRoot); dirExists(d) {
		return d
	}
	if d := filepath.Join(projectRoot, RecipesDirName); dirExists(d) {
		return d
	}
	return RecipesDir(projectRoot)
}

// dirExists 判断路径是否为存在的目录
func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// ResolveAssetDir 解析图片等资源目录（配置 asset_dir，默认 .flavor-vault/assets）。
// 维护者数据源检出优先使用 worktree / source 检出内的资源目录。
func ResolveAssetDir(projectRoot string, cfg *models.Config) string {
	base := ".flavor-vault/assets"
	if cfg != nil && strings.TrimSpace(cfg.AssetDir) != "" {
		base = cfg.AssetDir
	}
	dir := filepath.Join(projectRoot, filepath.FromSlash(base))
	if cfg != nil && cfg.Maintainer() {
		wtDir := filepath.Join(RecipesWorktree(projectRoot), filepath.FromSlash(base))
		if info, err := os.Stat(wtDir); err == nil && info.IsDir() {
			return wtDir
		}
		srcDir := SourceAssetDir(projectRoot, cfg)
		if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
			return srcDir
		}
	}
	return dir
}

// ConfigPath 返回配置文件路径
func ConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, ConfigName)
}

// HomeRoot 返回用户主目录（AI/安装方式各异时的稳定根目录）
func HomeRoot() string {
	d, err := os.UserHomeDir()
	if err != nil || d == "" {
		return "."
	}
	return d
}

// HomeConfigPath 返回用户主目录下的默认配置（~/.flavor-vault/config.yaml）
func HomeConfigPath() string {
	return filepath.Join(HomeRoot(), DirName, ConfigName)
}

// SaveConfig 将配置写回文件（原子写入，YAML 格式）
func SaveConfig(projectRoot string, cfg *models.Config) error {
	return SaveConfigAt(ConfigPath(projectRoot), cfg)
}

// SaveConfigAt 将配置写入指定路径（YAML 格式）
func SaveConfigAt(configPath string, cfg *models.Config) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0o644)
}

func bytesReader(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}

// ResolveOutputDir 将配置中的输出目录解析为绝对路径
func ResolveOutputDir(projectRoot string, cfg *models.Config) string {
	out := cfg.OutputDir
	if out == "" {
		out = "./dist"
	}
	if filepath.IsAbs(out) {
		return out
	}
	return filepath.Join(projectRoot, out)
}
