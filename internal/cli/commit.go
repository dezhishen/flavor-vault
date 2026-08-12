package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
	"flavor-vault/internal/utils"
)

// confirmCommit 在提交前预览菜谱与变更，请求用户确认（--yes 跳过；回车/空默认提交）。
// 返回 false 表示用户取消（调用方不应提交）。
// 供 fv add / edit / gh push --recipe 在 stageLocalAssets 之后、实际推送之前调用。
func confirmCommit(cmd *cobra.Command, r *models.Recipe, message string) (bool, error) {
	if yes, _ := cmd.Flags().GetBool("yes"); yes {
		return true, nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n—— 提交预览 ——\n")
	fmt.Fprintf(cmd.OutOrStdout(), "菜谱: %s（%s）\n", r.Name, r.ID)
	if message != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "提交说明: %s\n", message)
	}
	if n := localAssetCount(r); n > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "图片资源: %d 个（随菜谱一起提交到数据分支）\n", n)
	}
	pretty, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return false, err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "内容:\n%s\n\n", pretty)

	fmt.Fprint(cmd.OutOrStdout(), "确认提交? [Y/n] ")
	ok, err := confirmFromReader(bufio.NewReader(os.Stdin))
	if err != nil {
		return false, err
	}
	if !ok {
		fmt.Fprintln(cmd.OutOrStdout(), "✋ 已取消，未提交")
		return false, nil
	}
	return true, nil
}

// confirmFromReader 读取一行确认：空/回车/y/yes/是 → true（默认提交）；n/no/否 → false。
func confirmFromReader(r *bufio.Reader) (bool, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	ans := strings.ToLower(strings.TrimSpace(line))
	if ans == "n" || ans == "no" || ans == "否" {
		return false, nil
	}
	return true, nil
}

// localAssetCount 统计菜谱中非远程 URL 的资源引用数（待随菜谱上传的本地图片）。
func localAssetCount(r *models.Recipe) int {
	n := 0
	for _, ref := range r.AssetRefs() {
		if ref != "" && !utils.IsRemoteURL(ref) {
			n++
		}
	}
	return n
}
