package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/data"
	"flavor-vault/internal/plugins"
	"flavor-vault/internal/store"
	"flavor-vault/internal/utils"
)

// listItem 列表展示用的轻量条目
type listItem struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Tags        []string `json:"tags"`
	Kitchenware []string `json:"kitchenware"`
	Difficulty  int      `json:"difficulty"`
	TotalTime   int      `json:"total_time"`
}

func newListCmd() *cobra.Command {
	var (
		tag    string
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有菜谱，支持按标签过滤（支持远程 endpoint）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}

			var filtered []*listItem

			// 使用者模式：配置了 endpoint，直接读取部署好的 all.json（与 pages 同一套数据）
			if data.RemoteEndpoint(cfg) != "" {
				locator, remote := data.Locator(cfg, projectRoot, "all.json")
				raw, err := data.ReadJSON(locator, remote)
				if err != nil {
					return err
				}
				var summaries []plugins.RecipeSummary
				if err := json.Unmarshal(raw, &summaries); err != nil {
					return err
				}
				for _, s := range summaries {
					if tag != "" && !utils.Contains(s.Tags, tag) {
						continue
					}
					filtered = append(filtered, &listItem{
						ID: s.ID, Name: s.Name, Tags: s.Tags,
						Kitchenware: s.Kitchenware, Difficulty: s.Difficulty,
						TotalTime: s.PrepTime + s.CookTime,
					})
				}
			} else {
				// 维护者模式：读取本地（自有 + 外部数据源）菜谱
				res, err := store.LoadAllMulti(allRecipeDirs(cfg, projectRoot), store.LoadOptions{SkipInvalid: true})
				if err != nil {
					return err
				}
				for _, w := range res.Warnings {
					fmt.Fprintln(cmd.ErrOrStderr(), "⚠", w)
				}
				for _, r := range res.Recipes {
					if tag != "" && !utils.Contains(r.Tags, tag) {
						continue
					}
					filtered = append(filtered, &listItem{
						ID: r.ID, Name: r.Name, Tags: r.Tags,
						Kitchenware: r.Kitchenware, Difficulty: r.Stats.Difficulty,
						TotalTime: r.TotalTime(),
					})
				}
			}

			if jsonOut {
				data2, _ := json.MarshalIndent(filtered, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(data2))
				return nil
			}
			if len(filtered) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(无菜谱，运行 fv add 添加，或检查 endpoint)")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "共 %d 道菜谱:\n", len(filtered))
			for _, r := range filtered {
				fmt.Fprintf(cmd.OutOrStdout(), "  %-24s %-16s [%s] %d分钟 ★%d\n",
					r.ID, r.Name, strings.Join(r.Tags, ","), r.TotalTime, r.Difficulty)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "按标签过滤")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "以 JSON 数组输出")
	return cmd
}
