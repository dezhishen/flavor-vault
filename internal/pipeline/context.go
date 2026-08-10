package pipeline

import (
	"github.com/spf13/cobra"

	"flavor-vault/internal/cache"
	"flavor-vault/internal/models"
)

// Plugin 插件接口
type Plugin interface {
	// 插件唯一标识
	Name() string
	// 构建阶段执行（所有插件顺序执行）
	Build(ctx *BuildContext) error
	// 向根命令注册子命令（可选）
	RegisterCommands(root *cobra.Command) error
}

// BuildContext 构建上下文
type BuildContext struct {
	Recipes    []*models.Recipe          // 所有菜谱
	Config     *models.Config            // 全局配置
	ConfigPath string                    // 配置文件路径（用于依赖哈希）
	OutputDir  string                    // 输出根目录（如 ./dist）
	CacheRoot  string                    // 缓存根目录（如 .flavor-vault/cache）
	Force      bool                      // 是否强制重建
	Options    map[string]interface{}    // 额外参数
	Cache      *cache.CacheManager       // 缓存管理器
	DataDir    string                    // 输出 data 目录（OutputDir/data）
}

// NewBuildContext 构造构建上下文
func NewBuildContext(recipes []*models.Recipe, cfg *models.Config, outputDir, cacheRoot, configPath string, force bool) *BuildContext {
	dataDir := joinDataDir(outputDir)
	return &BuildContext{
		Recipes:    recipes,
		Config:     cfg,
		ConfigPath: configPath,
		OutputDir:  outputDir,
		CacheRoot:  cacheRoot,
		Force:      force,
		Options:    make(map[string]interface{}),
		Cache:      cache.NewCacheManager(cacheRoot),
		DataDir:    dataDir,
	}
}

func joinDataDir(outputDir string) string {
	return outputDir + "/data"
}
