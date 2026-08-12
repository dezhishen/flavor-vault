package models

import "strings"

// Config 全局配置结构，对应 .flavor-vault/config.yaml（可选）
// 同时标注 yaml 与 mapstructure 标签，确保 Viper 能正确映射 snake_case 键。
//
// 两种使用模式：
//   - 只读：只需 endpoint（+ cache/构建字段）
//   - 读写（编辑）：配置 github（菜谱仓库 + 权限），add/edit/rm 经 GitHub API 操作
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

	// 远程数据 endpoint（只读模式）。未配置时使用默认值；
	// 默认值可在构建时替换（cmd/build --endpoint / FV_ENDPOINT 写入产物 meta）。
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`

	// 作者信息（编辑提交时使用，fv init 强制采集）。
	// AI 助手环境下 git config 可能缺失，故写入配置供 add/edit/rm 提交使用。
	Author AuthorConfig `yaml:"author" mapstructure:"author"`

	// GitHub（读写模式）：菜谱数据仓库 + 权限。add/edit/rm 经 GitHub API
	// 直接操作该仓库/分支上的单文件，无需本地 clone。留空 = 只读。
	GitHub GitHubConfig `yaml:"github" mapstructure:"github"`
}

// AuthorConfig 作者信息（菜谱提交作者）
type AuthorConfig struct {
	Name  string `yaml:"name" mapstructure:"name"`
	Email string `yaml:"email" mapstructure:"email"`
}

// GitHubConfig 读写模式配置：菜谱数据仓库 + GitHub 权限
type GitHubConfig struct {
	// 访问令牌（也可用环境变量 GITHUB_TOKEN）
	Token string `yaml:"token" mapstructure:"token"`
	// 菜谱数据仓库："owner/repo" 或完整 git/HTTPS URL；留空 = 当前仓库（git remote）的独立分支
	Repo string `yaml:"repo" mapstructure:"repo"`
	// 菜谱数据分支（默认 recipes）
	Branch string `yaml:"branch" mapstructure:"branch"`
	// gh 操作默认分支（默认 main）
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

// Maintainer 是否为读写（编辑）模式：配置了菜谱数据仓库或分支
func (c *Config) Maintainer() bool {
	if c == nil {
		return false
	}
	return strings.TrimSpace(c.GitHub.Repo) != "" || strings.TrimSpace(c.GitHub.Branch) != ""
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
