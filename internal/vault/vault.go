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

// RecipesWorktree 返回菜谱独立分支的本地 worktree 目录（<root>/.recipes）
func RecipesWorktree(projectRoot string) string {
	return filepath.Join(projectRoot, ".recipes")
}

// ResolveRecipesDir 解析菜谱源目录：
// 配置了独立菜谱分支且本地 worktree 存在时，返回 worktree 下的菜谱目录；
// 否则返回默认的 <root>/.flavor-vault/recipes。
func ResolveRecipesDir(projectRoot string, cfg *models.Config) string {
	if cfg != nil && cfg.GitHub.RecipesBranch != "" {
		wt := RecipesWorktree(projectRoot)
		dir := filepath.Join(wt, DirName, RecipesDirName)
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return RecipesDir(projectRoot)
}

// ResolveAssetDir 解析图片等资源目录（配置 asset_dir，默认 .flavor-vault/assets）。
// 独立菜谱分支模式下优先使用 worktree 内的资源目录。
func ResolveAssetDir(projectRoot string, cfg *models.Config) string {
	base := ".flavor-vault/assets"
	if cfg != nil && strings.TrimSpace(cfg.AssetDir) != "" {
		base = cfg.AssetDir
	}
	dir := filepath.Join(projectRoot, filepath.FromSlash(base))
	if cfg != nil && cfg.GitHub.RecipesBranch != "" {
		wtDir := filepath.Join(RecipesWorktree(projectRoot), filepath.FromSlash(base))
		if info, err := os.Stat(wtDir); err == nil && info.IsDir() {
			return wtDir
		}
	}
	return dir
}

// ConfigPath 返回配置文件路径
func ConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, DirName, ConfigName)
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
