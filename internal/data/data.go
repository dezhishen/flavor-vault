package data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flavor-vault/internal/models"
	"flavor-vault/internal/vault"
)

// DefaultEndpoint 内置默认数据 endpoint（站点 data 目录）。
// CLI 未配置 endpoint 且本地无构建产物时使用；可在构建时替换（CI 注入 meta.json 的 endpoint 覆盖）。
const DefaultEndpoint = "https://fv.sdniu.top/data"

// RemoteEndpoint 返回配置的远程数据 endpoint（去尾斜杠）。
// 读取（非编辑）命令只依赖 endpoint：显式配置了就用远程。
func RemoteEndpoint(cfg *models.Config) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
}

// Locator 返回某数据文件（如 meta.json）的读取位置，优先级：
//  1. 显式 endpoint（--endpoint / 配置 / FV_ENDPOINT）→ 远程
//  2. 本地构建产物（dist/data/<file> 存在）→ 本地（离线/开发）
//  3. 构建时注入到本地 meta.json 的 endpoint → 远程
//  4. 内置默认 endpoint（DefaultEndpoint）→ 远程
//  5. 本地（文件不存在，读取时报"请先 fv build 或配置 endpoint"）
func Locator(cfg *models.Config, projectRoot, file string) (locator string, remote bool) {
	if e := RemoteEndpoint(cfg); e != "" {
		return e + "/" + file, true
	}
	outDir := vault.ResolveOutputDir(projectRoot, cfg)
	local := filepath.Join(outDir, "data", file)
	if info, err := os.Stat(local); err == nil && !info.IsDir() {
		return local, false
	}
	if e := DefaultEndpointFromMeta(cfg, projectRoot); e != "" {
		return e + "/" + file, true
	}
	return DefaultEndpoint + "/" + file, true
}

// DefaultEndpointFromMeta 返回构建时写入本地数据 meta.json 的 endpoint（构建时替换）。
func DefaultEndpointFromMeta(cfg *models.Config, projectRoot string) string {
	if cfg == nil {
		return ""
	}
	outDir := vault.ResolveOutputDir(projectRoot, cfg)
	raw, err := os.ReadFile(filepath.Join(outDir, "data", "meta.json"))
	if err != nil {
		return ""
	}
	var m struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(m.Endpoint), "/")
}

// ReadJSON 读取数据文件（远程 HTTP 或本地文件），返回原始字节
func ReadJSON(locator string, remote bool) ([]byte, error) {
	if remote {
		return httpGet(locator)
	}
	data, err := os.ReadFile(locator)
	if err != nil {
		return nil, fmt.Errorf("读取数据文件失败（请先 fv build，或配置 endpoint 从远端读取）: %w", err)
	}
	return data, nil
}

func httpGet(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求 endpoint %s 失败: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("endpoint %s 返回 HTTP %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 endpoint 响应失败: %w", err)
	}
	return data, nil
}
