package utils

import (
	"reflect"
	"testing"
)

func TestIntersectSorted(t *testing.T) {
	cases := []struct {
		name string
		a, b []string
		want []string
	}{
		{"empty", nil, nil, []string{}},
		{"disjoint", []string{"a", "b"}, []string{"c", "d"}, []string{}},
		{"partial", []string{"a", "b", "c"}, []string{"b", "c", "d"}, []string{"b", "c"}},
		{"duplicate in one", []string{"a", "a", "b"}, []string{"a", "b"}, []string{"a", "b"}},
		{"same", []string{"x"}, []string{"x"}, []string{"x"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := IntersectSorted(c.a, c.b)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("IntersectSorted(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestIntersect(t *testing.T) {
	got := Intersect(
		[]string{"a", "b", "c"},
		[]string{"b", "c", "d"},
		[]string{"c", "d", "e"},
	)
	want := []string{"c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Intersect = %v, want %v", got, want)
	}

	// 空列表导致整体为空
	got = Intersect([]string{"a"}, []string{})
	if len(got) != 0 {
		t.Errorf("Intersect with empty list should be empty, got %v", got)
	}

	// 无参数
	if len(Intersect()) != 0 {
		t.Error("Intersect() should return empty")
	}
}

func TestSortAndDedupe(t *testing.T) {
	got := SortAndDedupe([]string{"b", "a", "b", "c", "a"})
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SortAndDedupe = %v, want %v", got, want)
	}
	if len(SortAndDedupe(nil)) != 0 {
		t.Error("SortAndDedupe(nil) should be empty")
	}
}
