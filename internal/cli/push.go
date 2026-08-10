package cli

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// runGit 在项目根执行 git 命令，返回输出
func runGit(projectRoot string, args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = projectRoot
	out, err := c.CombinedOutput()
	return string(out), err
}

// newPushCmd 执行 git add + commit + push。
// 为避免冲突（非快进/并发），流程：加锁 → fetch → add → commit → 落后则 rebase → push。
func newPushCmd() *cobra.Command {
	var (
		noRebase bool
		force    bool
	)
	cmd := &cobra.Command{
		Use:   "push <message>",
		Short: "执行 git add + commit + push（自动 fetch/rebase 防冲突）",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, _, err := resolveProject(cmd)
			if err != nil {
				return err
			}
			message := strings.Join(args, " ")

			// 1. 推送锁（单一写者，避免并发推送）
			lock, err := acquireLock(projectRoot)
			if err != nil {
				return err
			}
			defer lock.Release()

			// 2. 先 fetch 远端（获取最新引用，用于检测落后）
			fmt.Fprintln(cmd.OutOrStdout(), "▶ git fetch")
			if out, err := runGit(projectRoot, "fetch"); err != nil && !strings.Contains(out, "Could not read from remote") {
				return fmt.Errorf("git fetch 失败: %w\n%s", err, out)
			}

			// 3. 本地提交
			fmt.Fprintln(cmd.OutOrStdout(), "▶ git add -A")
			if out, err := runGit(projectRoot, "add", "-A"); err != nil {
				return fmt.Errorf("git add 失败: %w\n%s", err, out)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "▶ git commit")
			out, err := runGit(projectRoot, "commit", "-m", message)
			if err != nil {
				if !strings.Contains(out, "nothing to commit") && !strings.Contains(out, "no changes added") {
					return fmt.Errorf("git commit 失败: %w\n%s", err, out)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "ℹ 没有需要提交的更改")
			} else {
				fmt.Fprint(cmd.OutOrStdout(), out)
			}

			// 4. 检测是否落后远端（非快进风险），落后则 rebase 或中止
			currentBranch := strings.TrimSpace(mustGit(projectRoot, "rev-parse", "--abbrev-ref", "HEAD"))
			behind, _ := runGit(projectRoot, "rev-list", "--count", "HEAD..@{upstream}")
			behind = strings.TrimSpace(behind)

			if behind != "0" && behind != "" {
				if noRebase {
					return fmt.Errorf("本地落后远端 %s 个提交（非快进），已中止以避免冲突；请先手动 git pull --rebase", behind)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "⚠ 本地落后远端 %s 个提交，自动执行 rebase 对齐...\n", behind)
				if out, err := runGit(projectRoot, "pull", "--rebase"); err != nil {
					return fmt.Errorf("git pull --rebase 失败（可能存在冲突，请手动解决）: %w\n%s", err, out)
				} else {
					fmt.Fprint(cmd.OutOrStdout(), out)
				}
			}

			// 5. 推送：默认非强推；--force 时用 --force-with-lease（带预期远端 SHA 校验）
			pushArgs := []string{"push", "-u", "origin", currentBranch}
			if force {
				pushArgs = append(pushArgs, "--force-with-lease")
				fmt.Fprintln(cmd.OutOrStdout(), "▶ git push --force-with-lease")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "▶ git push")
			}
			if out, err := runGit(projectRoot, pushArgs...); err != nil {
				return fmt.Errorf("git push 失败: %w\n%s", err, out)
			} else {
				fmt.Fprint(cmd.OutOrStdout(), out)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已推送 %s: %s\n", currentBranch, message)
			return nil
		},
	}
	cmd.Flags().BoolVar(&noRebase, "no-rebase", false, "落后远端时中止而非自动 rebase")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "需要覆盖时使用 --force-with-lease（带预期 SHA 校验）")
	return cmd
}

// mustGit 执行 git 命令，失败返回空串
func mustGit(projectRoot string, args ...string) string {
	out, err := runGit(projectRoot, args...)
	if err != nil {
		return ""
	}
	return out
}
