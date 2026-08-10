package pipeline

import (
	"testing"

	"github.com/spf13/cobra"

	"flavor-vault/internal/models"
)

type mockPlugin struct {
	name    string
	order   int
	fail    bool
	invoked *[]string
}

func (m *mockPlugin) Name() string { return m.name }

func (m *mockPlugin) Build(ctx *BuildContext) error {
	*m.invoked = append(*m.invoked, m.name)
	if m.fail {
		return &mockError{msg: m.name + " failed"}
	}
	return nil
}

func (m *mockPlugin) RegisterCommands(_ *cobra.Command) error { return nil }

type mockError struct{ msg string }

func (e *mockError) Error() string { return e.msg }

func TestSchedulerRunInOrder(t *testing.T) {
	var invoked []string
	s := NewScheduler(nil)
	s.AddPlugin(&mockPlugin{name: "a", invoked: &invoked})
	s.AddPlugin(&mockPlugin{name: "b", invoked: &invoked})
	s.AddPlugin(&mockPlugin{name: "c", invoked: &invoked})

	ctx := NewBuildContext(nil, models.DefaultConfig(), t.TempDir(), t.TempDir()+"/cache", "", true)
	if err := s.Run(ctx); err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c"}
	for i, v := range want {
		if invoked[i] != v {
			t.Fatalf("invoked order = %v, want %v", invoked, want)
		}
	}
}

func TestSchedulerStopsOnError(t *testing.T) {
	var invoked []string
	s := NewScheduler(nil)
	s.AddPlugin(&mockPlugin{name: "a", invoked: &invoked})
	s.AddPlugin(&mockPlugin{name: "b", fail: true, invoked: &invoked})
	s.AddPlugin(&mockPlugin{name: "c", invoked: &invoked})

	ctx := NewBuildContext(nil, models.DefaultConfig(), t.TempDir(), t.TempDir()+"/cache", "", true)
	if err := s.Run(ctx); err == nil {
		t.Fatal("expected error")
	}
	if len(invoked) != 2 {
		t.Fatalf("only first 2 plugins should run, invoked = %v", invoked)
	}
}
