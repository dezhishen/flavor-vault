package cli

import (
	"os"
	"path/filepath"
	"testing"

	"flavor-vault/internal/models"
)

// TestCoverReuseNoDuplicate 回归：cover 引用为扁平路径（assets 根）且 images/<id>/ 已有同名文件时，
// 归位应复用/覆盖该引用，而不是生成 -cover-2 重复文件（修复 U+FFFD 同源图片被再次复制）。
func TestCoverReuseNoDuplicate(t *testing.T) {
	root := t.TempDir()
	cfg := &models.Config{}
	// assets 根已有扁平 cover（本地源）
	os.MkdirAll(filepath.Join(root, ".flavor-vault/assets"), 0o755)
	if err := os.WriteFile(filepath.Join(root, ".flavor-vault/assets/超绝紫苏虾-cover.jpg"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	// images/<id>/ 已有此前归位的同名 cover
	d := filepath.Join(root, ".flavor-vault/assets/images/chao-jue-zi-su-xia")
	os.MkdirAll(d, 0o755)
	if err := os.WriteFile(filepath.Join(d, "超绝紫苏虾-cover.jpg"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := &models.Recipe{
		ID:       "chao-jue-zi-su-xia",
		Name:     "超绝紫苏虾",
		Versions: []models.Version{{Media: models.Media{Cover: "超绝紫苏虾-cover.jpg"}}},
	}
	n, err := stageLocalAssets(cfg, root, r)
	if err != nil {
		t.Fatalf("stageLocalAssets err: %v", err)
	}
	if r.Versions[0].Media.Cover != "images/chao-jue-zi-su-xia/超绝紫苏虾-cover.jpg" {
		t.Fatalf("应复用目标引用而非 -2: %s", r.Versions[0].Media.Cover)
	}
	if n != 1 {
		t.Fatalf("引用被归位（1 处变更），得到 %d", n)
	}
	// 不应产生 -cover-2 重复文件
	matches, _ := filepath.Glob(filepath.Join(d, "*cover-2*"))
	if len(matches) != 0 {
		t.Fatalf("不应生成 -2 重复文件: %v", matches)
	}
}

// TestStageGroupedRefIdempotent 回归：已分组引用本地存在时幂等原样（不重复复制、不 -2）
func TestStageGroupedRefIdempotent(t *testing.T) {
	root := t.TempDir()
	cfg := &models.Config{}
	d := filepath.Join(root, ".flavor-vault/assets/images/chao-jue-zi-su-xia")
	os.MkdirAll(d, 0o755)
	if err := os.WriteFile(filepath.Join(d, "超绝紫苏虾-cover.jpg"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &models.Recipe{
		ID:       "chao-jue-zi-su-xia",
		Name:     "超绝紫苏虾",
		Versions: []models.Version{{Media: models.Media{Cover: "images/chao-jue-zi-su-xia/超绝紫苏虾-cover.jpg"}}},
	}
	if n, err := stageLocalAssets(cfg, root, r); err != nil {
		t.Fatalf("err: %v", err)
	} else if n != 0 {
		t.Fatalf("已分组引用应幂等（0 变更），得到 %d", n)
	}
	if r.Versions[0].Media.Cover != "images/chao-jue-zi-su-xia/超绝紫苏虾-cover.jpg" {
		t.Fatalf("引用被改动: %s", r.Versions[0].Media.Cover)
	}
}
