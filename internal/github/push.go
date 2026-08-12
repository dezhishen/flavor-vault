package github

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v63/github"
)

// ErrNonFastForward 远端已被他人推进，拒绝覆盖（防冲突）
var ErrNonFastForward = errors.New("推送冲突：远端分支已更新（non-fast-forward），已中止以避免覆盖他人提交；请重新拉取后再推送")

// PushResult 推送结果
type PushResult struct {
	Branch    string
	CommitSHA string
	BaseSHA   string
	Message   string
}

// HeadSHA 返回分支当前 tip 的提交 SHA
func (c *Client) HeadSHA(ctx context.Context, branch string) (string, error) {
	ref, _, err := c.gh.Git.GetRef(ctx, c.Owner, c.Repo, "refs/heads/"+branch)
	if err != nil {
		if ghErr, ok := err.(*github.ErrorResponse); ok && ghErr.Response.StatusCode == http.StatusNotFound {
			return "", nil // 分支不存在
		}
		return "", err
	}
	if ref.Object == nil || ref.Object.SHA == nil {
		return "", fmt.Errorf("分支 %s 的引用对象为空", branch)
	}
	return ref.Object.GetSHA(), nil
}

// FastForwardPush 将 files 作为一次提交推送到分支，带快进守卫：
//  1. 读取远端分支 tip SHA（baseSHA），若分支不存在则基于默认分支创建；
//  2. 以 baseSHA 为父提交创建 blob/tree/commit；
//  3. 更新 ref 前再次校验远端 tip 未变（乐观锁），若已被他人推进则返回 ErrNonFastForward。
//
// 通过"单一写者 + 快进校验"避免与并发推送 / 远端新提交产生冲突。
func (c *Client) FastForwardPush(ctx context.Context, branch, message string, files map[string][]byte, author Author) (*PushResult, error) {
	// 1. 获取远端 tip
	baseSHA, err := c.HeadSHA(ctx, branch)
	if err != nil {
		return nil, err
	}

	// 分支不存在时，基于仓库默认分支 tip 创建（避免产生孤立分支）
	if baseSHA == "" {
		baseSHA, err = c.HeadSHA(ctx, DefaultBranch(""))
		if err != nil {
			return nil, err
		}
	}

	// 2. 构建 tree：base tree + 新文件
	baseTreeSHA, err := c.baseTreeSHA(ctx, baseSHA)
	if err != nil {
		return nil, err
	}

	entries := make([]*github.TreeEntry, 0, len(files))
	for path, content := range files {
		blob, err := c.createBlob(ctx, content)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &github.TreeEntry{
			Path: github.String(path),
			Mode: github.String("100644"),
			Type: github.String("blob"),
			SHA:  github.String(blob),
		})
	}

	newTree, _, err := c.gh.Git.CreateTree(ctx, c.Owner, c.Repo, baseTreeSHA, entries)
	if err != nil {
		return nil, fmt.Errorf("创建 tree 失败: %w", err)
	}

	// 3. 创建 commit
	parents := []string{}
	if baseSHA != "" {
		parents = append(parents, baseSHA)
	}
	commit := &github.Commit{
		Message: github.String(message),
		Tree:    &github.Tree{SHA: newTree.SHA},
		Parents: []*github.Commit{{SHA: github.String(baseSHA)}},
		Author:  &github.CommitAuthor{Name: github.String(author.Name), Email: github.String(author.Email)},
	}
	if baseSHA == "" {
		commit.Parents = nil
	}
	newCommit, _, err := c.gh.Git.CreateCommit(ctx, c.Owner, c.Repo, commit, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 commit 失败: %w", err)
	}

	// 4. 快进守卫：更新 ref 前校验远端 tip 未变
	current, err := c.HeadSHA(ctx, branch)
	if err != nil {
		return nil, err
	}
	if current != baseSHA {
		return nil, ErrNonFastForward
	}

	ref := &github.Reference{
		Ref:    github.String("refs/heads/" + branch),
		Object: &github.GitObject{SHA: newCommit.SHA},
	}
	if _, _, err := c.gh.Git.UpdateRef(ctx, c.Owner, c.Repo, ref, false); err != nil {
		// 422 = 冲突（并发推进），转为友好的冲突错误
		return nil, fmt.Errorf("%w（%v）", ErrNonFastForward, err)
	}

	return &PushResult{Branch: branch, CommitSHA: newCommit.GetSHA(), BaseSHA: baseSHA, Message: message}, nil
}

// baseTreeSHA 返回某提交的 tree SHA（提交不存在则返回空）
func (c *Client) baseTreeSHA(ctx context.Context, commitSHA string) (string, error) {
	if commitSHA == "" {
		return "", nil
	}
	commit, _, err := c.gh.Git.GetCommit(ctx, c.Owner, c.Repo, commitSHA)
	if err != nil {
		return "", fmt.Errorf("读取提交 %s 失败: %w", commitSHA, err)
	}
	if commit.Tree == nil || commit.Tree.SHA == nil {
		return "", fmt.Errorf("提交 %s 缺少 tree", commitSHA)
	}
	return commit.Tree.GetSHA(), nil
}

// createBlob 创建 blob，返回 SHA。
// 注意：必须显式 base64 编码再上传——go-github 的 JSON 序列化会把字符串里的
// 无效 UTF-8 字节替换为 U+FFFD，若直接 string(content)+encoding=utf-8，
// 二进制（JPEG 等）会被整体损坏；base64 内容为纯 ASCII，服务端按 base64 还原原始字节。
func (c *Client) createBlob(ctx context.Context, content []byte) (string, error) {
	blob := &github.Blob{
		Content:  github.String(base64.StdEncoding.EncodeToString(content)),
		Encoding: github.String("base64"),
	}
	b, _, err := c.gh.Git.CreateBlob(ctx, c.Owner, c.Repo, blob)
	if err != nil {
		return "", fmt.Errorf("创建 blob 失败: %w", err)
	}
	return b.GetSHA(), nil
}
