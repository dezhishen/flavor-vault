package cli

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
)

func TestLocalAssetCount(t *testing.T) {
	r := &models.Recipe{
		Media: models.Media{
			Cover:  "images/hong-shao-rou/红烧肉-cover.jpg",
			Images: []string{"https://x.com/a.png", ""},
		},
		Steps: []models.Step{{Order: 1, ImageRef: "images/hong-shao-rou/红烧肉-1-1.png"}},
	}
	if n := localAssetCount(r); n != 2 {
		t.Fatalf("期望 2 个本地资源（cover + 步骤图，远程与空引用排除），得到 %d", n)
	}
}

func TestConfirmFromReader(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"\n", true},    // 回车默认提交
		{"y\n", true},   //
		{"yes\n", true}, //
		{"是\n", true},   //
		{"Y\n", true},   //
		{"n\n", false},  //
		{"no\n", false}, //
		{"否\n", false},  //
		{"N\n", false},  //
	}
	for _, c := range cases {
		ok, err := confirmFromReader(bufio.NewReader(strings.NewReader(c.in)))
		if err != nil {
			t.Fatalf("confirmFromReader(%q) err: %v", c.in, err)
		}
		if ok != c.want {
			t.Fatalf("confirmFromReader(%q) = %v, want %v", c.in, ok, c.want)
		}
	}
}

// TestConfirmCommitPreviewAndCancel 验证 confirmCommit 输出预览并支持取消（n 不提交）。
func TestConfirmCommitPreviewAndCancel(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("yes", "y", false, "")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	r := &models.Recipe{
		ID:   "hong-shao-rou",
		Name: "红烧肉",
		Media: models.Media{
			Cover: "images/hong-shao-rou/红烧肉-cover.jpg",
		},
	}

	// 模拟 stdin 输入 n（取消）
	oldStdin := os.Stdin
	rd, w, _ := os.Pipe()
	os.Stdin = rd
	_, _ = w.WriteString("n\n")
	_ = w.Close()
	defer func() { os.Stdin = oldStdin }()

	ok, err := confirmCommit(cmd, r, "add: 红烧肉")
	if err != nil {
		t.Fatalf("confirmCommit err: %v", err)
	}
	if ok {
		t.Fatal("输入 n 应取消提交")
	}
	preview := out.String()
	for _, want := range []string{"提交预览", "红烧肉", "hong-shao-rou", "图片资源: 1 个", "已取消"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("预览输出缺少 %q:\n%s", want, preview)
		}
	}
}

// TestConfirmCommitYesSkip 验证 --yes 跳过确认直接提交。
func TestConfirmCommitYesSkip(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().BoolP("yes", "y", false, "")
	_ = cmd.Flags().Set("yes", "true")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	ok, err := confirmCommit(cmd, &models.Recipe{ID: "x", Name: "X"}, "")
	if err != nil {
		t.Fatalf("confirmCommit err: %v", err)
	}
	if !ok {
		t.Fatal("--yes 应跳过确认直接提交")
	}
	if strings.Contains(out.String(), "提交预览") {
		t.Fatal("--yes 不应输出预览")
	}
}
