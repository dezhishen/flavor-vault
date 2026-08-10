package pipeline

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Scheduler 调度器：顺序执行插件
type Scheduler struct {
	plugins []Plugin
	logger  io.Writer
}

// NewScheduler 创建调度器
func NewScheduler(logger io.Writer) *Scheduler {
	if logger == nil {
		logger = os.Stderr
	}
	return &Scheduler{logger: logger}
}

// AddPlugin 添加插件
func (s *Scheduler) AddPlugin(p Plugin) {
	s.plugins = append(s.plugins, p)
}

// Plugins 返回已注册插件列表
func (s *Scheduler) Plugins() []Plugin {
	return s.plugins
}

// Run 顺序执行所有插件
func (s *Scheduler) Run(ctx *BuildContext) error {
	for _, p := range s.plugins {
		fmt.Fprintf(s.logger, "▶ 插件 [%s] 开始...\n", p.Name())
		if err := p.Build(ctx); err != nil {
			return fmt.Errorf("插件 %s 执行失败: %w", p.Name(), err)
		}
		fmt.Fprintf(s.logger, "✔ 插件 [%s] 完成\n", p.Name())
	}
	return nil
}

// RegisterCommands 让所有插件注册子命令
func (s *Scheduler) RegisterCommands(root *cobra.Command) error {
	for _, p := range s.plugins {
		if err := p.RegisterCommands(root); err != nil {
			return fmt.Errorf("插件 %s 注册命令失败: %w", p.Name(), err)
		}
	}
	return nil
}

// EnsureDir 确保目录存在
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o755)
}

// WriteFile 写入文件（确保父目录存在）
func WriteFile(path string, data []byte) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
