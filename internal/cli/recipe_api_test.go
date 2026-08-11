package cli

import (
	"os"
	"path/filepath"
	"testing"

	"flavor-vault/internal/models"
)

// TestStageLocalAssets 验证本地图片暂存逻辑（修复 edit / add --json 图片未上传）：
// 本地图片复制到资源目录并更新引用；远程 URL 与分支已有资产（images/）原样保留；缺失本地图片报错。
func TestStageLocalAssets(t *testing.T) {
	root := t.TempDir()
	cfg := &models.Config{}

	r := &models.Recipe{Name: "红烧肉"}
	img := filepath.Join(root, "meat.png")
	if err := os.WriteFile(img, []byte("fake-png"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.Media.Cover = img
	r.Steps = append(r.Steps,
		models.Step{Order: 2, ImageRef: "https://example.com/a.png"},
		models.Step{Order: 3, ImageRef: "images/已有.png"},
	)

	n, err := stageLocalAssets(cfg, root, r)
	if err != nil {
		t.Fatalf("stageLocalAssets err: %v", err)
	}
	if n != 1 {
		t.Fatalf("期望暂存 1 个本地图片，得到 %d", n)
	}
	if r.Media.Cover != "images/红烧肉-cover.png" {
		t.Fatalf("cover 引用未按规范更新: %s", r.Media.Cover)
	}
	if _, err := os.Stat(filepath.Join(root, ".flavor-vault/assets/images/红烧肉-cover.png")); err != nil {
		t.Fatalf("暂存的图片文件不存在: %v", err)
	}
	// 远程 URL 与分支已有资产引用保持原样
	if r.Steps[0].ImageRef != "https://example.com/a.png" {
		t.Fatal("远程 URL 引用被改动")
	}
	if r.Steps[1].ImageRef != "images/已有.png" {
		t.Fatal("分支已有资产引用被改动")
	}

	// 缺失本地图片 → 报错
	r2 := &models.Recipe{Name: "测试", Media: models.Media{Cover: filepath.Join(root, "nope.png")}}
	if _, err := stageLocalAssets(cfg, root, r2); err == nil {
		t.Fatal("缺失本地图片应报错")
	}

	// 基路径前缀引用规范化：.flavor-vault/assets/封面.jpg → 封面.jpg（文件已在资源目录则原样保留）
	assetDir := filepath.Join(root, ".flavor-vault/assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "封面.jpg"), []byte("jpg"), 0o644); err != nil {
		t.Fatal(err)
	}
	r3 := &models.Recipe{Name: "紫苏虾", Media: models.Media{Cover: ".flavor-vault/assets/封面.jpg"}}
	if n, err := stageLocalAssets(cfg, root, r3); err != nil {
		t.Fatalf("规范化失败: %v", err)
	} else if n != 1 {
		t.Fatalf("引用被规范化（算 1 处变更），得到 %d", n)
	}
	if r3.Media.Cover != "封面.jpg" {
		t.Fatalf("未去掉 assetBase 前缀: %s", r3.Media.Cover)
	}
	if _, err := os.Stat(filepath.Join(assetDir, "封面.jpg")); err != nil {
		t.Fatalf("资源目录内的文件应保留: %v", err)
	}
}
