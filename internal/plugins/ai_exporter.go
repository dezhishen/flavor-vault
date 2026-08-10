package plugins

import (
	"bufio"
	"bytes"
	"encoding/json"
	"path/filepath"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
	"flavor-vault/internal/pipeline"
)

// AICorpusEntry AI 快照单条记录（JSON Lines 每行一条）
type AICorpusEntry struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Tags            []string `json:"tags"`
	Kitchenware     []string `json:"kitchenware"`
	MainIngredients []string `json:"main_ingredients"`
	PrepTime        int      `json:"prep_time"`
	CookTime        int      `json:"cook_time"`
	Difficulty      int      `json:"difficulty"`
}

// AIExporter 生成 AI 专用的精简快照（JSON Lines）
type AIExporter struct{}

// Name 插件标识
func (p *AIExporter) Name() string { return "ai_exporter" }

// RegisterCommands ai_exporter 不注册子命令
func (p *AIExporter) RegisterCommands(_ *cobra.Command) error { return nil }

// Build 生成 ai-corpus.json（仅当配置开启 ai_snapshot）
func (p *AIExporter) Build(ctx *pipeline.BuildContext) error {
	if !ctx.Config.AISnapshot {
		return nil
	}
	outPath := filepath.Join(ctx.DataDir, "ai-corpus.json")
	return cachedWrite(ctx, p.Name(), outPath, func() ([]byte, error) {
		var buf bytes.Buffer
		w := bufio.NewWriter(&buf)
		for _, r := range ctx.Recipes {
			line, err := json.Marshal(toAICorpusEntry(r))
			if err != nil {
				return nil, err
			}
			if _, err := w.WriteString(string(line) + "\n"); err != nil {
				return nil, err
			}
		}
		if err := w.Flush(); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	})
}

func toAICorpusEntry(r *models.Recipe) AICorpusEntry {
	return AICorpusEntry{
		ID:              r.ID,
		Name:            r.Name,
		Description:     r.Description,
		Tags:            r.Tags,
		Kitchenware:     r.Kitchenware,
		MainIngredients: r.MainIngredientNames(),
		PrepTime:        r.Stats.PrepTime,
		CookTime:        r.Stats.CookTime,
		Difficulty:      r.Stats.Difficulty,
	}
}
