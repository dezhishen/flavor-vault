package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"flavor-vault/internal/action"
	"flavor-vault/internal/models"
)

// getActionID 读取全局 --action-id 参数
func getActionID(cmd *cobra.Command) string {
	id, _ := cmd.Flags().GetString("action-id")
	return strings.TrimSpace(id)
}

// actionStoreFor 返回 action-id 对应的缓存 Store；未指定时返回 nil
func actionStoreFor(cmd *cobra.Command) *action.Store {
	if id := getActionID(cmd); id != "" {
		return action.New(id)
	}
	return nil
}

// loadCachedRecipe 从缓存恢复指定类型的菜谱草稿
func loadCachedRecipe(st *action.Store, wantAction string) (*models.Recipe, bool) {
	if st == nil || !st.Exists() {
		return nil, false
	}
	a, err := st.Load()
	if err != nil || a.Action != wantAction || a.Recipe == nil {
		return nil, false
	}
	return a.Recipe, true
}

// cacheRecipe 将菜谱草稿保存到动作缓存（用于失败后重试/续写）
func cacheRecipe(st *action.Store, act, targetID string, r *models.Recipe) error {
	if st == nil {
		return nil
	}
	a := &action.Action{
		Action:   act,
		TargetID: targetID,
		Recipe:   r,
		Status:   "pending",
	}
	return st.Save(a)
}

// failAndCache 校验失败时：若配置了 action-id 则缓存草稿，并提示重试路径
func failAndCache(cmd *cobra.Command, st *action.Store, act, targetID string, r *models.Recipe, err error) error {
	if st != nil {
		if cErr := cacheRecipe(st, act, targetID, r); cErr == nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ 操作未完成，参数已缓存到 %s\n", st.Path())
			fmt.Fprintf(cmd.ErrOrStderr(), "  ➜ 修正后以相同 action-id 重试即可继续（如 fv %s --action-id %s --json ...）\n", act, st.ID)
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "⚠ 缓存操作参数失败: %v\n", cErr)
		}
	}
	return err
}

// completeAction 动作无误完成后清除缓存，并输出提示
func completeAction(cmd *cobra.Command, st *action.Store) {
	if st == nil || !st.Exists() {
		return
	}
	if err := st.Clear(); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "⚠ 清除动作缓存失败: %v\n", err)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✔ 动作完成，已清除缓存 %s\n", st.Path())
}

// readJSONInput 读取 --json 参数：支持内联 JSON 或 @文件路径
func readJSONInput(raw string) ([]byte, error) {
	if strings.HasPrefix(raw, "@") {
		path := strings.TrimPrefix(raw, "@")
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("读取 JSON 文件 %s 失败: %w", path, err)
		}
		return data, nil
	}
	return []byte(raw), nil
}

// parseRecipeJSON 解析菜谱 JSON 到目标结构
func parseRecipeJSON(raw string, r *models.Recipe) error {
	data, err := readJSONInput(raw)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, r); err != nil {
		return fmt.Errorf("解析菜谱 JSON 失败: %w", err)
	}
	return nil
}

// now 时间戳辅助
func now() time.Time { return time.Now() }
