package data

import (
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

// RemoteEndpoint 返回配置的远程数据 endpoint（去尾斜杠）
func RemoteEndpoint(cfg *models.Config) string {
	if cfg != nil {
		return strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	}
	return ""
}

// Locator 返回某数据文件（如 meta.json）的读取位置。
// 配置了 endpoint 时返回远程 URL（与 pages 部署同一套数据）；否则返回本地 dist/data/<file>。
func Locator(cfg *models.Config, projectRoot, file string) (locator string, remote bool) {
	if e := RemoteEndpoint(cfg); e != "" {
		return e + "/" + file, true
	}
	outDir := vault.ResolveOutputDir(projectRoot, cfg)
	return filepath.Join(outDir, "data", file), false
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
