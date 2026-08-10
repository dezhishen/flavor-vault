package utils

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// FileHash 计算文件的 MD5 哈希
func FileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// StringHash 计算字符串的 MD5 哈希（用于简单依赖）
func StringHash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
