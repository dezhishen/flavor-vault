package models

import (
	"encoding/json"
	"testing"
)

// TestVersionsEffective 验证多版本与默认版本回退
func TestVersionsEffective(t *testing.T) {
	// 无 versions → 顶层字段作为单个默认版本
	r := &Recipe{Ingredients: Ingredients{Main: []Ingredient{{Name: "五花肉"}}}}
	vs := r.VersionsEffective()
	if len(vs) != 1 {
		t.Fatalf("期望 1 个默认版本，得到 %d", len(vs))
	}
	if len(vs[0].Ingredients.Main) != 1 {
		t.Fatalf("默认版本应含顶层食材")
	}

	// 有 versions → 使用 versions
	r2 := &Recipe{Versions: []Version{{Name: "经典版"}, {Name: "少油版"}}}
	if got := len(r2.VersionsEffective()); got != 2 {
		t.Fatalf("期望 2 个版本，得到 %d", got)
	}
}

// TestMainIngredientNamesAggregate 验证跨版本聚合主料（去重保序）
func TestMainIngredientNamesAggregate(t *testing.T) {
	r := &Recipe{Versions: []Version{
		{Ingredients: Ingredients{Main: []Ingredient{{Name: "五花肉"}, {Name: "冰糖"}}}},
		{Ingredients: Ingredients{Main: []Ingredient{{Name: "五花肉"}, {Name: "白萝卜"}}}},
	}}
	got := r.MainIngredientNames()
	want := []string{"五花肉", "冰糖", "白萝卜"}
	if len(got) != len(want) {
		t.Fatalf("聚合主料=%v，期望 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("聚合主料=%v，期望 %v", got, want)
		}
	}
}

// TestIngredientNamesAll 验证聚合主/配/可选 + 调料（含备选）
func TestIngredientNamesAll(t *testing.T) {
	r := &Recipe{Versions: []Version{{
		Ingredients: Ingredients{
			Main:     []Ingredient{{Name: "五花肉"}},
			Side:     []Ingredient{{Name: "冰糖"}},
			Optional: []Ingredient{{Name: "鹌鹑蛋"}},
		},
		Seasonings: []Seasoning{
			{Name: "香葱", Alternatives: []SeasoningOption{{Name: "香菜"}}},
			{Name: "料酒"},
		},
	}}}
	got := r.IngredientNamesAll()
	for _, want := range []string{"五花肉", "冰糖", "鹌鹑蛋", "香葱", "香菜", "料酒"} {
		found := false
		for _, g := range got {
			if g == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("缺少 %s，实际 %v", want, got)
		}
	}
}

// TestMarshalJSONVersions 验证序列化时 versions/seasonings/optional 空值输出 [] 而非 null
func TestMarshalJSONVersions(t *testing.T) {
	r := &Recipe{Name: "测试", Versions: []Version{{Name: "v1"}}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	vs, ok := m["versions"].([]interface{})
	if !ok || len(vs) != 1 {
		t.Fatalf("versions 解析异常: %v", m["versions"])
	}
	// 顶层 seasonings 应为 [] 而非 null
	if _, ok := m["seasonings"].([]interface{}); !ok {
		t.Fatalf("seasonings 应为数组，得到 %T", m["seasonings"])
	}
}
