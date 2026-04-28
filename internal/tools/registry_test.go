package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
)

type mockTool struct {
	nameVal string
}

func (m *mockTool) Name() string                                            { return m.nameVal }
func (m *mockTool) Description() string                                     { return "mock" }
func (m *mockTool) Parameters() json.RawMessage                             { return json.RawMessage(`{}`) }
func (m *mockTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return "", nil }

func TestRegistry_Filter(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockTool{nameVal: "read_file"})
	r.Register(&mockTool{nameVal: "list_files"})
	r.Register(&mockTool{nameVal: "grep_search"})
	r.Register(&mockTool{nameVal: "run_command"})

	tests := []struct {
		name        string
		allowed     []string
		wantNames   []string
		wantLen     int
		originalLen int
	}{
		{
			name:        "filter to subset",
			allowed:     []string{"read_file", "grep_search"},
			wantNames:   []string{"grep_search", "read_file"},
			wantLen:     2,
			originalLen: 4,
		},
		{
			name:        "filter with unknown names silently skipped",
			allowed:     []string{"read_file", "nonexistent"},
			wantNames:   []string{"read_file"},
			wantLen:     1,
			originalLen: 4,
		},
		{
			name:        "nil allowed returns copy of full registry",
			allowed:     nil,
			wantNames:   []string{"grep_search", "list_files", "read_file", "run_command"},
			wantLen:     4,
			originalLen: 4,
		},
		{
			name:        "empty allowed returns copy of full registry",
			allowed:     []string{},
			wantNames:   []string{"grep_search", "list_files", "read_file", "run_command"},
			wantLen:     4,
			originalLen: 4,
		},
		{
			name:        "all unknown returns empty registry",
			allowed:     []string{"unknown1", "unknown2"},
			wantNames:   []string{},
			wantLen:     0,
			originalLen: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := r.Filter(tt.allowed)

			got := filtered.List()
			if len(got) != tt.wantLen {
				t.Errorf("filtered registry len = %d, want %d", len(got), tt.wantLen)
			}

			for i, wantName := range tt.wantNames {
				if i >= len(got) {
					break
				}
				if got[i].Name() != wantName {
					t.Errorf("tool[%d].Name() = %q, want %q", i, got[i].Name(), wantName)
				}
			}

		// Verify original registry is untouched.
		original := r.List()
		if len(original) != tt.originalLen {
			t.Errorf("original registry len = %d, want %d", len(original), tt.originalLen)
		}
	})
	}
}

func TestRegistry_Deregister(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockTool{nameVal: "server1__tool_a"})
	r.Register(&mockTool{nameVal: "server1__tool_b"})
	r.Register(&mockTool{nameVal: "server2__tool_a"})
	r.Register(&mockTool{nameVal: "builtin_tool"})

	r.Deregister("server1")

	tools := r.List()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools after deregister, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}

	if names["server1__tool_a"] {
		t.Error("expected server1__tool_a to be deregistered")
	}
	if names["server1__tool_b"] {
		t.Error("expected server1__tool_b to be deregistered")
	}
	if !names["server2__tool_a"] {
		t.Error("expected server2__tool_a to remain")
	}
	if !names["builtin_tool"] {
		t.Error("expected builtin_tool to remain")
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func(n int) {
			defer wg.Done()
			r.Register(&mockTool{nameVal: fmt.Sprintf("tool_%d", n)})
		}(i)
		go func(n int) {
			defer wg.Done()
			r.Get(fmt.Sprintf("tool_%d", n))
		}(i)
		go func() {
			defer wg.Done()
			r.List()
		}()
	}
	wg.Wait()

	// All registrations should have succeeded.
	tools := r.List()
	if len(tools) != 100 {
		t.Errorf("expected 100 tools, got %d", len(tools))
	}
}

func TestRegistry_DeregisterPrefixEdgeCases(t *testing.T) {
	r := NewRegistry()
	r.Register(&mockTool{nameVal: "server1__tool"})
	r.Register(&mockTool{nameVal: "server10__tool"})
	r.Register(&mockTool{nameVal: "__tool"})

	r.Deregister("server1")

	tools := r.List()
	names := make(map[string]bool)
	for _, t := range tools {
		names[t.Name()] = true
	}
	if names["server1__tool"] {
		t.Error("expected server1__tool to be removed")
	}
	if !names["server10__tool"] {
		t.Error("expected server10__tool to remain")
	}
	if !names["__tool"] {
		t.Error("expected __tool to remain")
	}
}

func TestRegistry_ConcurrentDeregister(t *testing.T) {
	r := NewRegistry()
	for i := 0; i < 50; i++ {
		r.Register(&mockTool{nameVal: fmt.Sprintf("srv__tool%d", i)})
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Deregister("srv")
		}()
	}
	wg.Wait()

	tools := r.List()
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(tools))
	}
}
