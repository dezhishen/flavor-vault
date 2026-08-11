package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v63/github"
	"golang.org/x/oauth2"

	"flavor-vault/internal/models"
)

// Client GitHub API 客户端封装（go-github）
type Client struct {
	gh    *github.Client
	Owner string
	Repo  string
}

// NewClient 使用 token 创建客户端
func NewClient(token, owner, repo string) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("缺少 GitHub token（设置环境变量 GITHUB_TOKEN 或 config 的 github.token）")
	}
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("缺少仓库信息（设置 config 的 github.owner/repo，或配置 git remote origin）")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: strings.TrimSpace(token)})
	httpClient := oauth2.NewClient(context.Background(), ts)
	return &Client{gh: github.NewClient(httpClient), Owner: owner, Repo: repo}, nil
}

// NewClientFromConfig 从配置创建客户端（token 优先环境变量 GITHUB_TOKEN）；
// Owner/Repo 需在创建后由调用方设置（可能从 git remote 解析）。
func NewClientFromConfig(cfg *models.Config) (*Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		token = cfg.GitHub.Token
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("缺少 GitHub token（设置环境变量 GITHUB_TOKEN 或 config 的 github.token）")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: strings.TrimSpace(token)})
	httpClient := oauth2.NewClient(context.Background(), ts)
	return &Client{gh: github.NewClient(httpClient)}, nil
}

// DefaultBranch 返回分支名（配置优先，默认 main）
func DefaultBranch(cfgBranch string) string {
	if strings.TrimSpace(cfgBranch) != "" {
		return strings.TrimSpace(cfgBranch)
	}
	return "main"
}

// ResolveRepo 从 git remote origin 解析代码仓库的属主/仓库名
func ResolveRepo(projectRoot string) (owner, repo string, err error) {
	url := gitRemoteURL(projectRoot)
	o, r := parseRemoteURL(url)
	if o == "" || r == "" {
		return "", "", fmt.Errorf("无法解析仓库：请设置 git remote origin")
	}
	return o, r, nil
}

// gitRemoteURL 读取 git remote origin 的 URL
func gitRemoteURL(projectRoot string) string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// parseRemoteURL 从 remote URL 解析 owner/repo
// 支持 https://github.com/owner/repo.git 与 git@github.com:owner/repo.git
func parseRemoteURL(url string) (owner, repo string) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", ""
	}
	// 去掉 .git 后缀与尾斜杠
	url = strings.TrimSuffix(url, "/")
	url = strings.TrimSuffix(url, ".git")

	// git@github.com:owner/repo 形式
	if idx := strings.Index(url, ":"); idx > 0 && strings.Contains(url, "@") {
		url = url[idx+1:]
	} else if idx := strings.Index(url, "://"); idx >= 0 {
		url = url[idx+3:]
		// 去掉凭据 user@host 部分
		if at := strings.LastIndex(url, "@"); at >= 0 {
			url = url[at+1:]
		}
		// 去掉主机部分
		if slash := strings.Index(url, "/"); slash >= 0 {
			url = url[slash+1:]
		}
	}
	// 此时应为 owner/repo
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2], parts[len(parts)-1]
	}
	return "", ""
}

// ParseRepoSpec 解析仓库标识："owner/repo"、完整 git/HTTPS URL、git@host:owner/repo
func ParseRepoSpec(spec string) (owner, repo string, err error) {
	s := strings.TrimSpace(spec)
	if s == "" {
		return "", "", fmt.Errorf("仓库标识不能为空")
	}
	// 非 URL 的 "owner/repo" 形式
	if !strings.Contains(s, "://") && !strings.HasPrefix(s, "git@") && !strings.Contains(s, "@") {
		parts := strings.Split(strings.TrimRight(s, "/"), "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0], parts[1], nil
		}
	}
	o, r := parseRemoteURL(s)
	if o == "" || r == "" {
		return "", "", fmt.Errorf("无法解析仓库标识 %q（应为 owner/repo 或 URL）", spec)
	}
	return o, r, nil
}

// Author 提交作者信息
type Author struct {
	Name  string
	Email string
}

// ResolveAuthor 获取作者：优先参数，其次 git config
func ResolveAuthor(projectRoot, explicit string) (Author, error) {
	if explicit != "" {
		name, email, ok := parseAuthor(explicit)
		if ok {
			return Author{Name: name, Email: email}, nil
		}
		return Author{}, fmt.Errorf("作者格式应为 \"Name <email>\"")
	}
	name := gitConfig(projectRoot, "user.name")
	email := gitConfig(projectRoot, "user.email")
	if name == "" || email == "" {
		return Author{}, fmt.Errorf("无法确定作者：请用 --author \"Name <email>\" 或配置 git config user.name/user.email")
	}
	return Author{Name: name, Email: email}, nil
}

func parseAuthor(s string) (name, email string, ok bool) {
	lo := strings.Index(s, "<")
	hi := strings.Index(s, ">")
	if lo < 0 || hi <= lo {
		return "", "", false
	}
	return strings.TrimSpace(s[:lo]), strings.TrimSpace(s[lo+1 : hi]), true
}

func gitConfig(projectRoot, key string) string {
	out, err := exec.Command("git", "config", "--get", key).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
