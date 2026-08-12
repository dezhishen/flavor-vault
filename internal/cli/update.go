package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	ghc "flavor-vault/internal/github"
)

// updateRepo 默认发布仓库（构建时可用 -X 覆盖；如 fork 场景）
var updateRepo = "dezhishen/flavor-vault"

// newUpdateCmd 自更新 fv：从 GitHub Releases 下载当前平台二进制并替换。
// 公开仓库无需 token；支持 --check 仅检查、--version 指定版本、--repo 指定发布仓库。
func newUpdateCmd() *cobra.Command {
	var (
		checkOnly bool
		force     bool
		pre       bool
		repo      string
		targetVer string
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "自更新 fv 到 GitHub Releases 最新版（或指定版本）",
		Example: `  fv update                 # 更新到最新正式版
  fv update --check          # 仅检查是否有新版本
  fv update --pre            # 更新到最新预览版（含预发布，方便测试）
  fv update --version v0.1.0 # 更新到指定版本
  fv update --repo owner/repo`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			owner, r, err := ghc.ParseRepoSpec(repo)
			if err != nil {
				return err
			}
			cl := ghc.NewPublicClient(owner, r)

			cur := strings.TrimSpace(version)
			if cur == "" {
				cur = "dev"
			}

			// 1. 确定目标 Release
			var rel *ghc.Release
			if strings.TrimSpace(targetVer) != "" {
				t := strings.TrimPrefix(strings.TrimSpace(targetVer), "v")
				if _, _, ok := parseVersion(t); ok {
					rel, err = cl.ReleaseByTag(ctx, "v"+t)
				} else {
					rel, err = cl.ReleaseByTag(ctx, strings.TrimSpace(targetVer))
				}
			} else if pre {
				rel, err = latestInclPre(ctx, cl)
			} else {
				rel, err = cl.LatestRelease(ctx)
			}
			if err != nil {
				return fmt.Errorf("获取 Release 失败: %w", err)
			}
			tag := strings.TrimPrefix(rel.Tag, "v")

			fmt.Fprintf(cmd.OutOrStdout(), "当前版本: %s\n", cur)
			fmt.Fprintf(cmd.OutOrStdout(), "最新版本: %s\n", tag)
			if rel.Prerelease {
				fmt.Fprintln(cmd.OutOrStdout(), "（预览版）")
			}

			if checkOnly {
				if isNewer(cur, tag) {
					fmt.Fprintln(cmd.OutOrStdout(), "→ 有新版本，运行 fv update 更新")
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), "→ 已是最新")
				}
				return nil
			}

			if !isNewer(cur, tag) && !force {
				fmt.Fprintln(cmd.OutOrStdout(), "已是最新，无需更新（--force 可强制覆盖）")
				return nil
			}

			// 2. 找到当前平台资源
			asset := "fv-" + runtime.GOOS + "-" + runtime.GOARCH
			if runtime.GOOS == "windows" {
				asset += ".exe"
			}
			url := ghc.ReleaseAssetURL(rel, asset)
			if url == "" {
				// 兼容旧命名（无 GOARCH 后缀）
				url = ghc.ReleaseAssetURL(rel, "fv-"+runtime.GOOS)
				if url == "" {
					return fmt.Errorf("未找到资源 %s（当前平台 %s/%s）", asset, runtime.GOOS, runtime.GOARCH)
				}
			}

			// 3. 下载到可执行文件同目录（保证同盘，便于原子替换）
			exe, err := os.Executable()
			if err != nil {
				return fmt.Errorf("无法定位当前可执行文件: %w", err)
			}
			exeAbs, err := filepath.Abs(exe)
			if err != nil {
				exeAbs = exe
			}
			fmt.Fprintf(cmd.OutOrStdout(), "下载 %s ...\n", url)
			tmp, err := os.CreateTemp(filepath.Dir(exeAbs), ".fv-update-*")
			if err != nil {
				return err
			}
			tmpPath := tmp.Name()
			defer os.Remove(tmpPath)

			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("下载失败: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
			}
			if _, err := io.Copy(tmp, resp.Body); err != nil {
				return fmt.Errorf("写入临时文件失败: %w", err)
			}
			if err := tmp.Close(); err != nil {
				return err
			}
			if err := os.Chmod(tmpPath, 0o755); err != nil {
				return err
			}

			// 4. 替换（Unix 可直接覆盖；Windows 运行中无法覆盖 → 延迟替换）
			if err := os.Rename(tmpPath, exeAbs); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "✔ 已更新到 v%s，重启后生效\n", tag)
				return nil
			}
			newPath := exeAbs + ".new"
			if err := os.Rename(tmpPath, newPath); err != nil {
				return fmt.Errorf("替换失败：新版已存到 %s，请手动替换: %v", newPath, err)
			}
			if runtime.GOOS == "windows" {
				spawnWindowsReplace(newPath, exeAbs)
				fmt.Fprintf(cmd.OutOrStdout(), "✔ 已下载新版到 %s，正在后台替换（稍后重新运行 fv 生效）\n", newPath)
				return nil
			}
			return fmt.Errorf("替换失败：新版已存到 %s，请手动替换", newPath)
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "仅检查最新版本，不下载")
	cmd.Flags().BoolVar(&force, "force", false, "版本相同也强制覆盖")
	cmd.Flags().BoolVar(&pre, "pre", false, "更新到最新预览版（含预发布 Release，方便测试）")
	cmd.Flags().StringVar(&repo, "repo", updateRepo, "发布仓库（owner/repo）")
	cmd.Flags().StringVar(&targetVer, "version", "", "目标版本（默认最新正式版；--pre 时默认最新预览版）")
	return cmd
}

// latestInclPre 返回最高版本 Release（含预发布），供 --pre 提前测试预览版
func latestInclPre(ctx context.Context, cl *ghc.Client) (*ghc.Release, error) {
	rels, err := cl.ListReleases(ctx)
	if err != nil {
		return nil, err
	}
	if len(rels) == 0 {
		return nil, fmt.Errorf("仓库没有 Release")
	}
	best := rels[0]
	for _, rel := range rels[1:] {
		if isNewer(strings.TrimPrefix(best.Tag, "v"), strings.TrimPrefix(rel.Tag, "v")) {
			best = rel
		}
	}
	return best, nil
}

// spawnWindowsReplace 在 Windows 派发一个后台脚本，把 .new 覆盖回可执行文件
func spawnWindowsReplace(newPath, exePath string) {
	script := filepath.Join(filepath.Dir(exePath), ".fv-update.bat")
	content := "@echo off\r\n" +
		"ping 127.0.0.1 -n 2 >nul\r\n" +
		fmt.Sprintf("move /Y \"%s\" \"%s\" >nul 2>&1\r\n", newPath, exePath) +
		fmt.Sprintf("del \"%s\" >nul 2>&1\r\n", script) +
		"del \"%~f0\" >nul 2>&1\r\n"
	_ = os.WriteFile(script, []byte(content), 0o755)
	_ = exec.Command("cmd", "/c", "start", "", "/min", script).Start()
}

// parseVersion 拆分版本号：数字段 + 预发布标记（如 "0.0.1-beta.2" → [0,0,1], "beta.2"）
func parseVersion(v string) (nums []int, pre string, ok bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexByte(v, '-'); i >= 0 {
		pre = v[i+1:]
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) == 0 {
		return nil, "", false
	}
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return nil, "", false
		}
		nums = append(nums, n)
	}
	return nums, pre, true
}

// isNewer 判断 target 是否比 cur 更新（语义化比较；正式版 > 预发布）
func isNewer(cur, target string) bool {
	cn, cpre, cok := parseVersion(cur)
	tn, tpre, tok := parseVersion(target)
	if !cok || !tok {
		return false
	}
	max := len(cn)
	if len(tn) > max {
		max = len(tn)
	}
	for i := 0; i < max; i++ {
		var a, b int
		if i < len(cn) {
			a = cn[i]
		}
		if i < len(tn) {
			b = tn[i]
		}
		if a != b {
			return b > a
		}
	}
	// 数字相同：正式版比预发布新
	if cpre != "" && tpre == "" {
		return true
	}
	if cpre == "" && tpre != "" {
		return false
	}
	return tpre != cpre
}
