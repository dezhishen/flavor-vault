package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v63/github"
)

// TestCreateBlobBase64Roundtrip 验证 createBlob 以 base64 上传二进制且无损。
// 回归：旧实现 string(content)+encoding=utf-8，go-github JSON 序列化会把无效
// UTF-8 字节替换成 U+FFFD，导致 JPEG 等二进制文件整体损坏。
func TestCreateBlobBase64Roundtrip(t *testing.T) {
	var reqBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/repos/o/r/git/blobs" {
			_ = json.NewDecoder(r.Body).Decode(&reqBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"sha":"abc123","url":"https://api.github.com/repos/o/r/git/blobs/abc123"}`))
			return
		}
		http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL + "/")
	gh := github.NewClient(nil)
	gh.BaseURL = u
	gh.UploadURL = u
	c := &Client{gh: gh, Owner: "o", Repo: "r"}

	// 模拟 JPEG 头 + 大量非 ASCII 字节（此前会被 UTF-8 化损坏）
	raw := append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00},
		bytes.Repeat([]byte{0xFE, 0xFD, 0x80, 0x00, 0xFF}, 256)...)

	if _, err := c.createBlob(context.Background(), raw); err != nil {
		t.Fatalf("createBlob: %v", err)
	}
	if reqBody == nil {
		t.Fatal("未捕获 CreateBlob 请求")
	}
	if enc, _ := reqBody["encoding"].(string); enc != "base64" {
		t.Fatalf("encoding = %q, want base64", enc)
	}
	content, _ := reqBody["content"].(string)
	decoded, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		t.Fatalf("content 不是合法 base64: %v", err)
	}
	if !bytes.Equal(decoded, raw) {
		t.Fatalf("roundtrip 不一致：got %d bytes, want %d bytes", len(decoded), len(raw))
	}
}
