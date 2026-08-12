package cli

import (
	"testing"

	"flavor-vault/internal/models"
)

// TestApplyEditPatch 验证 --json 编辑补丁：单版本菜谱直接更新顶层（保持单版本）；多版本默认编辑第一个版本；versions 整体替换
func TestApplyEditPatch(t *testing.T) {
	// 1) 单版本菜谱（无 versions）→ 版本字段补丁直接更新顶层，保持单版本结构（不迁移成多版本）
	r := &models.Recipe{
		Name:  "红烧肉",
		Stats: models.Stats{PrepTime: 10, CookTime: 60, Difficulty: 2},
		Steps: []models.Step{{Order: 1, Description: "焯水"}},
		Media: models.Media{Cover: "images/x.png"},
	}
	if err := applyEditPatch(`{"stats":{"difficulty":4}}`, r); err != nil {
		t.Fatalf("applyEditPatch err: %v", err)
	}
	if len(r.Versions) != 0 {
		t.Fatalf("单版本菜谱不应被迁移成多版本，得到 %d 个版本", len(r.Versions))
	}
	if r.Stats.Difficulty != 4 || r.Stats.PrepTime != 10 || r.Stats.CookTime != 60 {
		t.Fatalf("顶层 stats 未正确合并: %+v", r.Stats)
	}
	if len(r.Steps) != 1 || r.Media.Cover != "images/x.png" {
		t.Fatalf("顶层内容未保留: %+v", r)
	}

	// 1b) 单版本菜谱只传 steps（用户的步骤图场景）→ 顶层 steps 更新，其余保留
	r1 := &models.Recipe{
		Name:  "红烧肉",
		Stats: models.Stats{Difficulty: 2},
		Steps: []models.Step{{Order: 1, Description: "旧步骤"}},
		Media: models.Media{Cover: "images/x.png"},
	}
	if err := applyEditPatch(`{"steps":[{"order":1,"description":"新步骤","image_ref":"images/hong-shao-rou/红烧肉-1-1.png"}]}`, r1); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(r1.Versions) != 0 || len(r1.Steps) != 1 || r1.Steps[0].Description != "新步骤" {
		t.Fatalf("单版本 steps 应更新顶层且不迁移: %+v", r1)
	}
	if r1.Media.Cover != "images/x.png" || r1.Stats.Difficulty != 2 {
		t.Fatalf("未补丁字段应保留: %+v", r1)
	}

	// 2) 多版本 → 默认编辑第一个版本，其他版本不动
	r2 := &models.Recipe{Versions: []models.Version{
		{Name: "经典版", Stats: models.Stats{Difficulty: 3}},
		{Name: "少油版", Stats: models.Stats{Difficulty: 2}},
	}}
	if err := applyEditPatch(`{"stats":{"prep_time":20}}`, r2); err != nil {
		t.Fatalf("err: %v", err)
	}
	if r2.Versions[0].Stats.PrepTime != 20 || r2.Versions[0].Stats.Difficulty != 3 {
		t.Fatalf("第一个版本未合并: %+v", r2.Versions[0].Stats)
	}
	if r2.Versions[1].Stats.Difficulty != 2 || r2.Versions[1].Stats.PrepTime != 0 {
		t.Fatalf("第二个版本被误改: %+v", r2.Versions[1].Stats)
	}

	// 3) 补丁含 versions → 整体替换 + 顶层内容清空（内容在 versions）
	r3 := &models.Recipe{
		Versions: []models.Version{{Name: "旧"}},
		Stats:    models.Stats{Difficulty: 3},
		Steps:    []models.Step{{Order: 1, Description: "旧"}},
	}
	if err := applyEditPatch(`{"versions":[{"name":"新版","steps":[{"order":1,"description":"s"}],"ingredients":{"main":[{"name":"a"}]},"stats":{"difficulty":1}}]}`, r3); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(r3.Versions) != 1 || r3.Versions[0].Name != "新版" {
		t.Fatalf("versions 未整体替换: %+v", r3.Versions)
	}
	if len(r3.Steps) != 0 || r3.Stats.Difficulty != 0 || len(r3.Ingredients.Main) != 0 {
		t.Fatalf("补丁含 versions 时顶层内容应清空: %+v", r3)
	}
}
