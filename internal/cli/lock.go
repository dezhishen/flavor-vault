package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// pushLock 基于独占锁文件的推送互斥锁（单一写者原则，避免并发推送冲突）
type pushLock struct {
	path string
}

// acquireLock 获取推送锁；锁文件存在且过期（>10 分钟）时视为残留并接管
func acquireLock(projectRoot string) (*pushLock, error) {
	dir := filepath.Join(projectRoot, ".flavor-vault")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "push.lock")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		// 残留锁检测
		if info, serr := os.Stat(path); serr == nil && time.Since(info.ModTime()) > 10*time.Minute {
			_ = os.Remove(path)
			f, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		}
		if err != nil {
			return nil, fmt.Errorf("已有推送正在进行（锁文件 %s），请稍后重试；确认无其他推送后可删除该锁文件", path)
		}
	}
	_, _ = f.WriteString(time.Now().Format(time.RFC3339))
	_ = f.Close()
	fmt.Fprintf(os.Stderr, "🔒 已获取推送锁 %s\n", path)
	return &pushLock{path: path}, nil
}

// Release 释放推送锁
func (l *pushLock) Release() {
	if l == nil || l.path == "" {
		return
	}
	_ = os.Remove(l.path)
}
