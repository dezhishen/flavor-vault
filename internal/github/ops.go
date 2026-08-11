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

// CurrentUser 返回基于 token 的认证用户信息（GET /user）。
// 用于在无 git config / 配置作者时，自动从 GitHub 账户推导提交作者。
func (c *Client) CurrentUser(ctx context.Context) (*github.User, error) {
	u, _, err := c.gh.Users.Get(ctx, "")
	if err != nil {
		return nil, err
	}
	return u, nil
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

// ListDir 返回某分支指定目录下的文件名列表（只读，目录不存在返回空列表）
func (c *Client) ListDir(ctx context.Context, branch, dir string) ([]string, error) {
	_, entries, resp, err := c.gh.Repositories.GetContents(ctx, c.Owner, c.Repo, dir,
		&github.RepositoryContentGetOptions{Ref: branch})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.GetType() == "file" {
			names = append(names, e.GetName())
		}
	}
	return names, nil
}

// GetFile 返回某分支指定文件的原始内容与 SHA（只读；文件不存在返回 (nil, "", nil)）
func (c *Client) GetFile(ctx context.Context, branch, path string) ([]byte, string, error) {
	f, _, resp, err := c.gh.Repositories.GetContents(ctx, c.Owner, c.Repo, path,
		&github.RepositoryContentGetOptions{Ref: branch})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, "", nil
		}
		return nil, "", err
	}
	raw, err := f.GetContent()
	if err != nil {
		return nil, "", err
	}
	return []byte(raw), f.GetSHA(), nil
}

// DeleteFile 提交删除某分支上的文件（SHA 校验天然防并发覆盖）
func (c *Client) DeleteFile(ctx context.Context, branch, path, sha, message string, author Author) error {
	_, _, err := c.gh.Repositories.DeleteFile(ctx, c.Owner, c.Repo, path,
		&github.RepositoryContentFileOptions{
			Message: github.String(message),
			SHA:     github.String(sha),
			Branch:  github.String(branch),
			Author:  &github.CommitAuthor{Name: github.String(author.Name), Email: github.String(author.Email)},
		})
	if err != nil {
		return fmt.Errorf("删除 %s 失败: %w", path, err)
	}
	return nil
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

// LatestRelease 返回最新正式 Release（GET /releases/latest；不含预发布）
func (c *Client) LatestRelease(ctx context.Context) (*Release, error) {
	rel, _, err := c.gh.Repositories.GetLatestRelease(ctx, c.Owner, c.Repo)
	if err != nil {
		return nil, err
	}
	return &Release{Tag: rel.GetTagName(), Assets: rel.Assets}, nil
}

// ReleaseByTag 返回指定 tag 的 Release
func (c *Client) ReleaseByTag(ctx context.Context, tag string) (*Release, error) {
	rel, _, err := c.gh.Repositories.GetReleaseByTag(ctx, c.Owner, c.Repo, tag)
	if err != nil {
		return nil, err
	}
	return &Release{Tag: rel.GetTagName(), Assets: rel.Assets}, nil
}

// ReleaseAssetURL 在 Release 资源中查找指定名称的下载地址
func ReleaseAssetURL(rel *Release, name string) string {
	if rel == nil {
		return ""
	}
	for _, a := range rel.Assets {
		if a.GetName() == name {
			return a.GetBrowserDownloadURL()
		}
	}
	return ""
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
