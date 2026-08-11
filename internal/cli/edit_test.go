package cli

import (
	"testing"

	"flavor-vault/internal/models"
)

// TestApplyEditPatch 验证 --json 编辑补丁：旧结构自动转单版本；多版本默认编辑第一个版本；versions 整体替换
func TestApplyEditPatch(t *testing.T) {
	// 1) 旧结构（顶层内容）→ 补丁后转为单版本结构
	r := &models.Recipe{
		Name:  "红烧肉",
		Stats: models.Stats{PrepTime: 10, CookTime: 60, Difficulty: 2},
		Steps: []models.Step{{Order: 1, Description: "焯水"}},
		Media: models.Media{Cover: "images/x.png"},
	}
	if err := applyEditPatch(`{"stats":{"difficulty":4}}`, r); err != nil {
		t.Fatalf("applyEditPatch err: %v", err)
	}
	if len(r.Versions) != 1 {
		t.Fatalf("应转为 1 个版本，得到 %d", len(r.Versions))
	}
	v := r.Versions[0]
	if v.Stats.Difficulty != 4 || v.Stats.PrepTime != 10 || v.Stats.CookTime != 60 {
		t.Fatalf("版本 stats 未正确合并: %+v", v.Stats)
	}
	if len(v.Steps) != 1 || v.Media.Cover != "images/x.png" {
		t.Fatalf("版本内容未保留: %+v", v)
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

	// 3) 补丁含 versions → 整体替换
	r3 := &models.Recipe{Versions: []models.Version{{Name: "旧"}}}
	if err := applyEditPatch(`{"versions":[{"name":"新版","steps":[{"order":1,"description":"s"}],"ingredients":{"main":[{"name":"a"}]},"stats":{"difficulty":1}}]}`, r3); err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(r3.Versions) != 1 || r3.Versions[0].Name != "新版" {
		t.Fatalf("versions 未整体替换: %+v", r3.Versions)
	}
}
