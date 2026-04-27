package tools

import (
	"context"
	"encoding/json"
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
