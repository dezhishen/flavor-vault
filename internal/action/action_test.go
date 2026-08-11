package action

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"flavor-vault/internal/models"
)

func TestActionStoreLifecycle(t *testing.T) {
	dir := t.TempDir()
	st := NewWithDir("abc123", dir)

	if st.Exists() {
		t.Fatal("should not exist initially")
	}

	r := &models.Recipe{ID: "hong-shao-rou", Name: "红烧肉"}
	a := &Action{Action: "add", TargetID: r.ID, Recipe: r}
	if err := st.Save(a); err != nil {
		t.Fatal(err)
	}
	if !st.Exists() {
		t.Fatal("should exist after Save")
	}
	expected := filepath.Join(dir, "action-abc123.json")
	if st.Path() != expected {
		t.Fatalf("path = %s, want %s", st.Path(), expected)
	}

	loaded, err := st.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ActionID != "abc123" {
		t.Errorf("action id = %s, want abc123", loaded.ActionID)
	}
	if loaded.Action != "add" || loaded.Recipe == nil || loaded.Recipe.Name != "红烧肉" {
		t.Errorf("unexpected action: %+v", loaded)
	}
	if loaded.Status != "pending" {
		t.Errorf("status = %s, want pending", loaded.Status)
	}
	if loaded.CreatedAt.IsZero() || loaded.UpdatedAt.IsZero() {
		t.Error("timestamps should be set")
	}

	if err := st.Clear(); err != nil {
		t.Fatal(err)
	}
	if st.Exists() {
		t.Fatal("should not exist after Clear")
	}
}

func TestActionList(t *testing.T) {
	dir := t.TempDir()
	old := os.Getenv("FV_ACTION_DIR")
	t.Setenv("FV_ACTION_DIR", dir)
	defer func() { _ = os.Setenv("FV_ACTION_DIR", old) }()

	s1 := NewWithDir("one", dir)
	s2 := NewWithDir("two", dir)
	_ = s1.Save(&Action{Action: "add", TargetID: "a", Recipe: &models.Recipe{ID: "a", Name: "A"}})
	time.Sleep(10 * time.Millisecond)
	_ = s2.Save(&Action{Action: "rm", TargetID: "b"})

	actions, err := List()
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	// 按更新时间倒序：two 应在 one 之前
	if actions[0].ActionID != "two" {
		t.Errorf("first action = %s, want two (most recent)", actions[0].ActionID)
	}

	// 空目录返回空列表
	empty := t.TempDir()
	_ = os.Setenv("FV_ACTION_DIR", empty)
	actions, err = List()
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 0 {
		t.Errorf("expected empty list, got %d", len(actions))
	}
}

func TestStoreRejectsEmptyID(t *testing.T) {
	st := New("")
	if err := st.Save(&Action{Action: "add"}); err == nil {
		t.Fatal("expected error for empty action-id")
	}
}

func TestDefaultDir(t *testing.T) {
	old := os.Getenv("FV_ACTION_DIR")
	t.Setenv("FV_ACTION_DIR", "")
	defer func() { _ = os.Setenv("FV_ACTION_DIR", old) }()
	want := filepath.Join(os.TempDir(), DefaultDirName)
	if got := Dir(); got != want {
		t.Fatalf("Dir() = %s, want %s", got, want)
	}
}
