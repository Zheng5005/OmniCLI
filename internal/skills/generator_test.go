package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGenerator_DetectVariables(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
		want   []string
	}{
		{
			name:   "no variables",
			prompt: "You are a helpful assistant.",
			want:   nil,
		},
		{
			name:   "single variable",
			prompt: "You are an expert in {{topic}}.",
			want:   []string{"topic"},
		},
		{
			name:   "multiple variables",
			prompt: "Write docs for {{project_name}} focusing on {{focus_area}}.",
			want:   []string{"focus_area", "project_name"},
		},
		{
			name:   "duplicate variables",
			prompt: "{{name}} and {{name}} again",
			want:   []string{"name"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewGenerator("test")
			g.Skill.SystemPrompt = tt.prompt
			got := g.DetectVariables()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DetectVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerator_Save(t *testing.T) {
	dir := t.TempDir()
	g := NewGenerator("test-skill")
	g.OutputPath = filepath.Join(dir, "test-skill.json")
	g.Skill.DisplayName = "Test Skill"
	g.Skill.SystemPrompt = "You are a test expert for {{project}}."
	g.Skill.Variables = map[string]string{"project": "The project name"}
	g.Skill.Tools = []string{"read_file", "list_files"}
	g.Skill.SafeList = []string{"^ls\\s", "^git\\sstatus"}
	g.Skill.AutoExecuteSafe = true

	if err := g.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(g.OutputPath); err != nil {
		t.Fatalf("expected file to exist after Save(): %v", err)
	}

	// Verify content.
	data, err := os.ReadFile(g.OutputPath)
	if err != nil {
		t.Fatalf("failed to read saved file: %v", err)
	}

	var got Skill
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal saved file: %v", err)
	}

	if got.Name != "test-skill" {
		t.Errorf("Name = %q, want %q", got.Name, "test-skill")
	}
	if got.DisplayName != "Test Skill" {
		t.Errorf("DisplayName = %q, want %q", got.DisplayName, "Test Skill")
	}
	if got.SystemPrompt != "You are a test expert for {{project}}." {
		t.Errorf("SystemPrompt = %q, want %q", got.SystemPrompt, "You are a test expert for {{project}}.")
	}
	if !reflect.DeepEqual(got.Variables, map[string]string{"project": "The project name"}) {
		t.Errorf("Variables = %v, want %v", got.Variables, map[string]string{"project": "The project name"})
	}
	if !reflect.DeepEqual(got.Tools, []string{"read_file", "list_files"}) {
		t.Errorf("Tools = %v, want %v", got.Tools, []string{"read_file", "list_files"})
	}
	if !reflect.DeepEqual(got.SafeList, []string{"^ls\\s", "^git\\sstatus"}) {
		t.Errorf("SafeList = %v, want %v", got.SafeList, []string{"^ls\\s", "^git\\sstatus"})
	}
	if got.AutoExecuteSafe != true {
		t.Errorf("AutoExecuteSafe = %v, want true", got.AutoExecuteSafe)
	}
}

func TestGenerator_Save_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	g := NewGenerator("test")
	g.OutputPath = filepath.Join(dir, "nested", "deep", "test.json")
	g.Skill.SystemPrompt = "prompt"

	if err := g.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := os.Stat(g.OutputPath); err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
}

func TestGenerator_Exists(t *testing.T) {
	dir := t.TempDir()
	g := NewGenerator("test")
	g.OutputPath = filepath.Join(dir, "test.json")

	if g.Exists() {
		t.Error("Exists() = true, want false before saving")
	}

	g.Skill.SystemPrompt = "prompt"
	if err := g.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if !g.Exists() {
		t.Error("Exists() = false, want true after saving")
	}
}
