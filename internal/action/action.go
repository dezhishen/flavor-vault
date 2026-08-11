package action

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"flavor-vault/internal/models"
)

// 默认缓存子目录名：<系统临时目录>/flavor-vaults（跨平台；可用 FV_ACTION_DIR 覆盖）
const DefaultDirName = "flavor-vaults"

// Action 缓存的单次操作
type Action struct {
	Action    string         `json:"action"`              // add | edit | rm
	ActionID  string         `json:"action_id"`           // 操作 ID
	TargetID  string         `json:"target_id,omitempty"` // 目标菜谱 ID（edit/rm；add 为生成的 id）
	Recipe    *models.Recipe `json:"recipe,omitempty"`    // 菜谱草稿（add/edit）
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	Status    string         `json:"status"` // pending | done
}

// Store 基于 action-id 的操作缓存
type Store struct {
	ID  string
	Dir string
}

// New 创建指定 action-id 的缓存（使用默认目录）
func New(id string) *Store {
	return &Store{ID: id, Dir: Dir()}
}

// NewWithDir 使用自定义目录
func NewWithDir(id, dir string) *Store {
	return &Store{ID: id, Dir: dir}
}

// Dir 返回缓存根目录：默认 <系统临时目录>/flavor-vaults（Linux /tmp、Windows %TEMP%），
// 可用环境变量 FV_ACTION_DIR 覆盖。
func Dir() string {
	if d := os.Getenv("FV_ACTION_DIR"); strings.TrimSpace(d) != "" {
		return strings.TrimSpace(d)
	}
	return filepath.Join(os.TempDir(), DefaultDirName)
}

// Path 返回该 action-id 对应的缓存文件路径
func (s *Store) Path() string {
	return filepath.Join(s.Dir, fmt.Sprintf("action-%s.json", s.ID))
}

// Exists 判断缓存文件是否存在
func (s *Store) Exists() bool {
	_, err := os.Stat(s.Path())
	return err == nil
}

// Load 读取缓存的操作
func (s *Store) Load() (*Action, error) {
	data, err := os.ReadFile(s.Path())
	if err != nil {
		return nil, err
	}
	var a Action
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, fmt.Errorf("解析动作缓存 %s 失败: %w", s.Path(), err)
	}
	return &a, nil
}

// Save 保存操作到缓存（原子写入）
func (s *Store) Save(a *Action) error {
	if s.ID == "" {
		return fmt.Errorf("action-id 不能为空")
	}
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("创建动作缓存目录失败: %w", err)
	}
	now := time.Now()
	if a.CreatedAt.IsZero() {
		a.CreatedAt = now
	}
	a.UpdatedAt = now
	a.ActionID = s.ID
	if a.Status == "" {
		a.Status = "pending"
	}
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.Path() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path())
}

// Clear 清除缓存
func (s *Store) Clear() error {
	if err := os.Remove(s.Path()); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// List 列出所有缓存的操作（按文件修改时间倒序）
func List() ([]*Action, error) {
	dir := Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Action{}, nil
		}
		return nil, err
	}
	var out []*Action
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "action-") || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(strings.TrimPrefix(e.Name(), "action-"), ".json")
		st := &Store{ID: id, Dir: dir}
		if a, err := st.Load(); err == nil {
			out = append(out, a)
		}
	}
	// 按更新时间倒序
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].UpdatedAt.After(out[j-1].UpdatedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}
