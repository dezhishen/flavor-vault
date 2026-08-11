package models

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
	// 外部菜谱数据源（可引用的外部仓库，供维护者聚合 / 使用者浏览）
	Sources []SourceConfig `yaml:"sources" mapstructure:"sources"`
	// 远程数据 endpoint（与 pages 部署同一套 data/ 数据）。
	// 设置后 list/filter/show/ask/stats 改为从该 URL 读取，适合"只查找"的使用者。
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint"`
	// GitHub 集成（fv gh / fv push）
	GitHub GitHubConfig `yaml:"github" mapstructure:"github"`
}

// SourceConfig 外部菜谱数据源：引用本项目外部的菜谱仓库
type SourceConfig struct {
	Name   string `yaml:"name" mapstructure:"name"`
	Repo   string `yaml:"repo" mapstructure:"repo"`
	Branch string `yaml:"branch" mapstructure:"branch"`
}

// GitHubConfig GitHub 客户端配置
type GitHubConfig struct {
	// 访问令牌（也可用环境变量 GITHUB_TOKEN）
	Token string `yaml:"token" mapstructure:"token"`
	// 仓库属主（默认从 git remote 推断）
	Owner string `yaml:"owner" mapstructure:"owner"`
	// 仓库名（默认从 git remote 推断）
	Repo string `yaml:"repo" mapstructure:"repo"`
	// 默认分支（默认 main）
	DefaultBranch string `yaml:"default_branch" mapstructure:"default_branch"`
	// 菜谱独立分支（如 "recipes"）：为空表示菜谱与代码同在默认分支。
	// 非空时，菜谱数据提交到该分支，构建/CRUD 从该分支的本地 worktree 读取。
	RecipesBranch string `yaml:"recipes_branch" mapstructure:"recipes_branch"`
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
