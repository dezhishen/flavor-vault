package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	ghc "flavor-vault/internal/github"
)

// newGhCmd 基于 go-github 的 GitHub 客户端命令组。
// 设计上避免冲突：只读查询 + 追加式操作（PR/Release/Workflow），
// 以及带"快进守卫"的 API 推送。
func newGhCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gh",
		Short: "通过 GitHub API 操作仓库（status/pr/release/workflow/push）",
	}
	cmd.AddCommand(
		newGhPushCmd(),
		newGhStatusCmd(),
		newGhPRCmd(),
		newGhReleaseCmd(),
		newGhWorkflowCmd(),
	)
	return cmd
}

// ghClient 组装 GitHub 客户端：token 来自环境变量/配置，owner/repo 来自配置或 git remote
func ghClient(cmd *cobra.Command) (*ghc.Client, string, error) {
	cfg, projectRoot, _, err := loadProjectConfig(cmd)
	if err != nil {
		return nil, "", err
	}
	cl, err := ghc.NewClientFromConfig(cfg)
	if err != nil {
		return nil, "", err
	}
	owner, repo, err := ghc.ResolveRepo(projectRoot, cfg.GitHub.Owner, cfg.GitHub.Repo)
	if err != nil {
		return nil, "", err
	}
	cl.Owner, cl.Repo = owner, repo
	return cl, projectRoot, nil
}

// ghBranch 解析分支：flag 优先，其次配置，默认 main
func ghBranch(cfgBranch, flag string) string {
	if strings.TrimSpace(flag) != "" {
		return strings.TrimSpace(flag)
	}
	return ghc.DefaultBranch(cfgBranch)
}

// ---------------------------------------------------------------------------

func newGhPushCmd() *cobra.Command {
	var (
		branchFlag string
		dirFlag    string
		authorFlag string
		ignoreFlag string
	)
	cmd := &cobra.Command{
		Use:   "push <message>",
		Short: "通过 GitHub API 推送文件（快进守卫，避免覆盖冲突）",
		Example: `  fv gh push "添加红烧肉"                      # 推送整个仓库（忽略缓存/构建产物）
  fv gh push "发布站点" --dir dist --branch gh-pages  # 推送构建产物到 gh-pages`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			cl, projectRoot, err := ghClient(cmd)
			if err != nil {
				return err
			}

			branch := ghBranch(cfg.GitHub.DefaultBranch, branchFlag)
			message := strings.Join(args, " ")

			// 收集要推送的文件
			src := dirFlag
			if src == "" {
				src = projectRoot
			}
			files, err := ghc.CollectFiles(src, splitCSV(ignoreFlag))
			if err != nil {
				return err
			}
			if len(files) == 0 {
				return fmt.Errorf("没有可推送的文件（目录 %s 为空或全部被忽略）", src)
			}

			// 推送锁（避免并发推送）
			lock, err := acquireLock(projectRoot)
			if err != nil {
				return err
			}
			defer lock.Release()

			// 作者信息
			author, err := ghc.ResolveAuthor(projectRoot, authorFlag)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "▶ 推送 %d 个文件到 %s/%s@%s ...\n", len(files), cl.Owner, cl.Repo, branch)
			res, err := cl.FastForwardPush(context.Background(), branch, message, files, author)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已推送 commit %s（父 %s）: %s\n", res.CommitSHA, shortSHA(res.BaseSHA), res.Message)
			return nil
		},
	}
	cmd.Flags().StringVar(&branchFlag, "branch", "", "目标分支（默认配置或 main）")
	cmd.Flags().StringVar(&dirFlag, "dir", "", "仅推送指定目录（如 dist）")
	cmd.Flags().StringVar(&authorFlag, "author", "", "作者 \"Name <email>\"（默认取 git config）")
	cmd.Flags().StringVar(&ignoreFlag, "ignore", "", "额外忽略的路径（逗号分隔）")
	return cmd
}

func newGhStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看仓库信息与最新提交/CI 状态（只读）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			cl, _, err := ghClient(cmd)
			if err != nil {
				return err
			}
			ctx := context.Background()

			repo, err := cl.GetRepo(ctx)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "仓库: %s/%s\n", repo.Owner, repo.Repo)
			fmt.Fprintf(cmd.OutOrStdout(), "默认分支: %s\n", repo.DefaultBranch)

			branch := ghBranch(cfg.GitHub.DefaultBranch, "")
			head, err := cl.HeadSHA(ctx, branch)
			if err != nil {
				return err
			}
			if head == "" {
				fmt.Fprintf(cmd.OutOrStdout(), "分支 %s 不存在\n", branch)
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "分支 %s tip: %s\n", branch, shortSHA(head))

			if state, err := cl.CombinedStatus(ctx, branch); err == nil {
				fmt.Fprintf(cmd.OutOrStdout(), "CI 状态: %s\n", state)
			}
			return nil
		},
	}
}

func newGhPRCmd() *cobra.Command {
	var (
		title string
		head  string
		base  string
		body  string
	)
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "创建 Pull Request（追加式，不写分支，无 ref 冲突）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := ghClient(cmd)
			if err != nil {
				return err
			}
			if title == "" || head == "" {
				return fmt.Errorf("--title 与 --head 为必填")
			}
			if base == "" {
				base = "main"
			}
			pr, err := cl.CreatePR(context.Background(), title, head, base, body)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), ghc.DescribePR(pr))
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "PR 标题（必填）")
	cmd.Flags().StringVar(&head, "head", "", "源分支（必填）")
	cmd.Flags().StringVar(&base, "base", "main", "目标分支")
	cmd.Flags().StringVar(&body, "body", "", "PR 描述")
	return cmd
}

func newGhReleaseCmd() *cobra.Command {
	var (
		tag        string
		name       string
		notes      string
		prerelease bool
	)
	cmd := &cobra.Command{
		Use:   "release",
		Short: "创建 Release（追加式）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := ghClient(cmd)
			if err != nil {
				return err
			}
			if tag == "" {
				return fmt.Errorf("--tag 为必填")
			}
			if name == "" {
				name = tag
			}
			rel, err := cl.CreateRelease(context.Background(), tag, name, notes, prerelease)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已创建 Release %s: %s\n", rel.GetTagName(), rel.GetHTMLURL())
			return nil
		},
	}
	cmd.Flags().StringVar(&tag, "tag", "", "标签（必填）")
	cmd.Flags().StringVar(&name, "name", "", "名称（默认取标签）")
	cmd.Flags().StringVar(&notes, "notes", "", "发布说明")
	cmd.Flags().BoolVar(&prerelease, "prerelease", false, "标记为预发布")
	return cmd
}

func newGhWorkflowCmd() *cobra.Command {
	var (
		workflow string
		ref      string
		inputs   []string
	)
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "触发 workflow_dispatch（追加式，不写分支 ref）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, _, err := loadProjectConfig(cmd)
			if err != nil {
				return err
			}
			cl, _, err := ghClient(cmd)
			if err != nil {
				return err
			}
			if workflow == "" {
				return fmt.Errorf("--workflow（工作流文件名）为必填")
			}
			branch := ghBranch(cfg.GitHub.DefaultBranch, ref)
			parsed := make(map[string]any, len(inputs))
			for _, kv := range inputs {
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					return fmt.Errorf("--input 格式应为 k=v，收到 %q", kv)
				}
				parsed[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
			if err := cl.DispatchWorkflow(context.Background(), workflow, branch, parsed); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✔ 已触发工作流 %s @ %s\n", workflow, branch)
			return nil
		},
	}
	cmd.Flags().StringVar(&workflow, "workflow", "", "工作流文件名（如 deploy.yml，必填）")
	cmd.Flags().StringVar(&ref, "ref", "", "目标分支（默认配置或 main）")
	cmd.Flags().StringArrayVar(&inputs, "input", nil, "输入参数 k=v（可多次指定）")
	return cmd
}

// splitCSV 按逗号拆分并去空白
func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// shortSHA 截断 SHA 便于展示
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
