package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v63/github"
)

// RepoInfo 仓库摘要
type RepoInfo struct {
	Owner         string
	Repo          string
	DefaultBranch string
	Description   string
}

// GetRepo 获取仓库信息（只读，无冲突风险）
func (c *Client) GetRepo(ctx context.Context) (*RepoInfo, error) {
	repo, _, err := c.gh.Repositories.Get(ctx, c.Owner, c.Repo)
	if err != nil {
		return nil, err
	}
	return &RepoInfo{
		Owner:         c.Owner,
		Repo:          c.Repo,
		DefaultBranch: repo.GetDefaultBranch(),
		Description:   repo.GetDescription(),
	}, nil
}

// CombinedStatus 返回某 ref 的合并 CI 状态（只读）
func (c *Client) CombinedStatus(ctx context.Context, ref string) (string, error) {
	st, _, err := c.gh.Repositories.GetCombinedStatus(ctx, c.Owner, c.Repo, ref, nil)
	if err != nil {
		return "", err
	}
	return st.GetState(), nil
}

// FileExists 判断某分支上指定路径的文件是否存在（只读）
func (c *Client) FileExists(ctx context.Context, branch, path string) (bool, error) {
	_, _, resp, err := c.gh.Repositories.GetContents(ctx, c.Owner, c.Repo, path,
		&github.RepositoryContentGetOptions{Ref: branch})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreatePR 创建 Pull Request（追加式，不写分支，天然无 ref 冲突）
func (c *Client) CreatePR(ctx context.Context, title, head, base, body string) (*github.PullRequest, error) {
	pr, _, err := c.gh.PullRequests.Create(ctx, c.Owner, c.Repo, &github.NewPullRequest{
		Title: github.String(title),
		Head:  github.String(head),
		Base:  github.String(base),
		Body:  github.String(body),
	})
	if err != nil {
		return nil, err
	}
	return pr, nil
}

// CreateRelease 创建 Release（追加式）
func (c *Client) CreateRelease(ctx context.Context, tag, name, notes string, prerelease bool) (*github.RepositoryRelease, error) {
	rel, _, err := c.gh.Repositories.CreateRelease(ctx, c.Owner, c.Repo, &github.RepositoryRelease{
		TagName:    github.String(tag),
		Name:       github.String(name),
		Body:       github.String(notes),
		Prerelease: github.Bool(prerelease),
	})
	if err != nil {
		return nil, err
	}
	return rel, nil
}

// DispatchWorkflow 触发 workflow_dispatch（追加式，不写分支 ref）
func (c *Client) DispatchWorkflow(ctx context.Context, workflow, ref string, inputs map[string]any) error {
	_, err := c.gh.Actions.CreateWorkflowDispatchEventByFileName(ctx, c.Owner, c.Repo, workflow,
		github.CreateWorkflowDispatchEventRequest{
			Ref:    ref,
			Inputs: inputs,
		})
	return err
}

// DescribePR 格式化 PR 摘要
func DescribePR(pr *github.PullRequest) string {
	if pr == nil {
		return ""
	}
	return fmt.Sprintf("#%d %s (%s → %s) %s", pr.GetNumber(), pr.GetTitle(), pr.GetHead().GetRef(), pr.GetBase().GetRef(), pr.GetHTMLURL())
}
