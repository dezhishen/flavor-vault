package models

import "strings"

// Config 全局配置结构，对应 .flavor-vault/config.yaml
// 同时标注 yaml 与 mapstructure 标签，确保 Viper 能正确映射 snake_case 键
type Config struct {
	// 缓存配置
	Cache CacheConfig `yaml:"cache" mapstructure:"cache"`
	// 构建输出目录（默认为 ./dist）
	OutputDir string `yaml:"output_dir" mapstructure:"output_dir"`
	// 是否生成 AI 快照
	AISnapshot bool `yaml:"ai_snapshot" mapstructure:"ai_snapshot"`
	// 插件配置（每个插件可覆盖 TTL）
	Plugins map[string]PluginConfig `yaml:"plugins" mapstructure:"plugins"`
	// 图片等资源目录（相对项目根，默认 .flavor-vault/assets）
	AssetDir string `yaml:"asset_dir" mapstructure:"asset_dir"`

	// 菜谱数据源（GitHub，唯一）。仅维护者配置：
	// CLI 的增删改/推送只作用于该数据源，绝不写入程序代码所在分支。
	// 留空 = 未配置（只读/使用者模式，通过 Endpoint 读取数据）。
	Source SourceConfig `yaml:"source" mapstructure:"source"`

	// 远程数据 endpoint（使用者模式）。
	// 未配置时使用默认值；默认值可在构建时替换（fv build --endpoint / FV_ENDPOINT 写入产物 meta）。
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`

	// GitHub 集成（fv gh / fv push）
	GitHub GitHubConfig `yaml:"github" mapstructure:"github"`
}

// SourceConfig 唯一菜谱数据源（GitHub 仓库，与代码隔离）
type SourceConfig struct {
	// 仓库标识："owner/repo" 或完整 git/HTTPS URL。
	// 留空且设置了 Branch 时表示"当前仓库的独立数据分支"。
	Repo string `yaml:"repo" mapstructure:"repo"`
	// 数据源分支（默认 recipes）
	Branch string `yaml:"branch" mapstructure:"branch"`
}

// GitHubConfig GitHub 客户端配置
type GitHubConfig struct {
	// 访问令牌（也可用环境变量 GITHUB_TOKEN）
	Token string `yaml:"token" mapstructure:"token"`
	// 代码仓库属主（默认从 git remote 推断；gh status/pr/release/workflow 用）
	Owner string `yaml:"owner" mapstructure:"owner"`
	// 代码仓库名（默认从 git remote 推断）
	Repo string `yaml:"repo" mapstructure:"repo"`
	// 默认分支（默认 main）
	DefaultBranch string `yaml:"default_branch" mapstructure:"default_branch"`
	// 推送前是否自动 fetch + rebase（fv push 防非快进冲突，默认 true）
	AutoRebase bool `yaml:"auto_rebase" mapstructure:"auto_rebase"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Enabled    bool           `yaml:"enabled" mapstructure:"enabled"`
	TTLSeconds int            `yaml:"ttl_seconds" mapstructure:"ttl_seconds"`
	PluginTTLs map[string]int `yaml:"plugins" mapstructure:"plugins"`
}

// PluginConfig 插件级配置
type PluginConfig struct {
	TTLSeconds int `yaml:"ttl_seconds" mapstructure:"ttl_seconds"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Cache: CacheConfig{
			Enabled:    true,
			TTLSeconds: 86400,
			PluginTTLs: map[string]int{
				"facet_indexer": 3600,
			},
		},
		OutputDir:  "./dist",
		AISnapshot: true,
		AssetDir:   ".flavor-vault/assets",
		GitHub: GitHubConfig{
			DefaultBranch: "main",
			AutoRebase:    true,
		},
	}
}

// Maintainer 是否为维护者模式（配置了菜谱数据源）。
// 维护者模式以本地数据源为准，不使用远程 endpoint 读取。
func (c *Config) Maintainer() bool {
	if c == nil {
		return false
	}
	return strings.TrimSpace(c.Source.Repo) != "" || strings.TrimSpace(c.Source.Branch) != ""
}

// PluginTTL 返回某插件的 TTL（秒），优先插件级配置，其次全局
func (c *Config) PluginTTL(pluginName string) int {
	if c.Cache.PluginTTLs != nil {
		if ttl, ok := c.Cache.PluginTTLs[pluginName]; ok {
			return ttl
		}
	}
	if c.Plugins != nil {
		if pc, ok := c.Plugins[pluginName]; ok && pc.TTLSeconds > 0 {
			return pc.TTLSeconds
		}
	}
	return c.Cache.TTLSeconds
}

// CacheEnabled 缓存总开关
func (c *Config) CacheEnabled() bool {
	return c.Cache.Enabled
}
