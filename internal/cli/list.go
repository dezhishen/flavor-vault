package cli

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/store"
	"flavor-vault/internal/utils"
)

func newListCmd() *cobra.Command {
	var (
		tag    string
		jsonOut bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有菜谱，支持按标签过滤",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, projectRoot, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			res, err := store.LoadAll(recipesDir(cfg, projectRoot), store.LoadOptions{SkipInvalid: true})
			if err != nil {
				return err
			}
			for _, w := range res.Warnings {
				fmt.Fprintln(cmd.ErrOrStderr(), "⚠", w)
			}

			var filtered []*struct {
				ID          string   `json:"id"`
				Name        string   `json:"name"`
				Tags        []string `json:"tags"`
				Kitchenware []string `json:"kitchenware"`
				Difficulty  int      `json:"difficulty"`
				TotalTime   int      `json:"total_time"`
			}
			for _, r := range res.Recipes {
				if tag != "" && !utils.Contains(r.Tags, tag) {
					continue
				}
				filtered = append(filtered, &struct {
					ID          string   `json:"id"`
					Name        string   `json:"name"`
					Tags        []string `json:"tags"`
					Kitchenware []string `json:"kitchenware"`
					Difficulty  int      `json:"difficulty"`
					TotalTime   int      `json:"total_time"`
				}{
					ID:          r.ID,
					Name:        r.Name,
					Tags:        r.Tags,
					Kitchenware: r.Kitchenware,
					Difficulty:  r.Stats.Difficulty,
					TotalTime:   r.TotalTime(),
				})
			}

			if jsonOut {
				data, _ := json.MarshalIndent(filtered, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(data))
				return nil
			}
			if len(filtered) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "(无菜谱，运行 fv add 添加)")
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
