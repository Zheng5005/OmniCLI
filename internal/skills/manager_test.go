package skills

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestManager_Get(t *testing.T) {
	// Setup temp directories.
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	// Create a project skill.
	projectSkill := `{"name":"project-skill","display_name":"Project Skill","system_prompt":"project prompt","tools":["read_file"]}`
	if err := os.WriteFile(filepath.Join(projectDir, "project-skill.json"), []byte(projectSkill), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a global skill.
	globalSkill := `{"name":"global-skill","display_name":"Global Skill","system_prompt":"global prompt","tools":["list_files"]}`
	if err := os.WriteFile(filepath.Join(globalDir, "global-skill.json"), []byte(globalSkill), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create an overridden skill (exists in both).
	overrideSkill := `{"name":"override-skill","display_name":"Project Override","system_prompt":"project version","tools":["grep_search"]}`
	if err := os.WriteFile(filepath.Join(projectDir, "override-skill.json"), []byte(overrideSkill), 0o644); err != nil {
		t.Fatal(err)
	}
	globalOverride := `{"name":"override-skill","display_name":"Global Override","system_prompt":"global version","tools":["read_file"]}`
	if err := os.WriteFile(filepath.Join(globalDir, "override-skill.json"), []byte(globalOverride), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		projectDir: projectDir,
		globalDir:  globalDir,
		cache:      make(map[string]*Skill),
	}

	tests := []struct {
		name        string
		skillName   string
		wantName    string
		wantDisplay string
		wantErr     bool
	}{
		{
			name:        "project skill",
			skillName:   "project-skill",
			wantName:    "project-skill",
			wantDisplay: "Project Skill",
			wantErr:     false,
		},
		{
			name:        "global skill",
			skillName:   "global-skill",
			wantName:    "global-skill",
			wantDisplay: "Global Skill",
			wantErr:     false,
		},
		{
			name:        "project overrides global",
			skillName:   "override-skill",
			wantName:    "override-skill",
			wantDisplay: "Project Override",
			wantErr:     false,
		},
		{
			name:      "not found",
			skillName: "nonexistent",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := m.Get(tt.skillName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.DisplayName != tt.wantDisplay {
				t.Errorf("DisplayName = %q, want %q", got.DisplayName, tt.wantDisplay)
			}
		})
	}
}

func TestManager_Get_CachesResult(t *testing.T) {
	projectDir := t.TempDir()
	skillContent := `{"name":"cached","display_name":"Cached","system_prompt":"prompt","tools":["read_file"]}`
	if err := os.WriteFile(filepath.Join(projectDir, "cached.json"), []byte(skillContent), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		projectDir: projectDir,
		cache:      make(map[string]*Skill),
	}

	// First fetch loads from disk.
	s1, err := m.Get("cached")
	if err != nil {
		t.Fatalf("first Get error: %v", err)
	}

	// Second fetch should return cached instance.
	s2, err := m.Get("cached")
	if err != nil {
		t.Fatalf("second Get error: %v", err)
	}

	if s1 != s2 {
		t.Error("expected cached skill instance to be reused")
	}
}

func TestManager_List(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	// Project skills.
	if err := os.WriteFile(filepath.Join(projectDir, "alpha.json"), []byte(`{"name":"alpha","system_prompt":"p","tools":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "beta.json"), []byte(`{"name":"beta","system_prompt":"p","tools":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Global skills.
	if err := os.WriteFile(filepath.Join(globalDir, "beta.json"), []byte(`{"name":"beta","system_prompt":"g","tools":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "gamma.json"), []byte(`{"name":"gamma","system_prompt":"g","tools":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		projectDir: projectDir,
		globalDir:  globalDir,
		cache:      make(map[string]*Skill),
	}

	got, err := m.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}

	want := []string{"alpha", "beta", "gamma"}
	if !sortedEqual(got, want) {
		t.Errorf("List() = %v, want %v", got, want)
	}
}

func TestManager_List_EmptyDirs(t *testing.T) {
	// Use non-existent directories.
	m := &Manager{
		projectDir: "/nonexistent/project",
		globalDir:  "/nonexistent/global",
		cache:      make(map[string]*Skill),
	}

	got, err := m.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty list, got %v", got)
	}
}

func TestManager_List_ProjectOnly(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "only.json"), []byte(`{"name":"only","system_prompt":"p","tools":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manager{
		projectDir: projectDir,
		globalDir:  "/nonexistent/global",
		cache:      make(map[string]*Skill),
	}

	got, err := m.List()
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	want := []string{"only"}
	if !sortedEqual(got, want) {
		t.Errorf("List() = %v, want %v", got, want)
	}
}

func TestDefaultDirs(t *testing.T) {
	projectDir, globalDir := DefaultDirs()
	if projectDir != ".omnicli/skills/" {
		t.Errorf("projectDir = %q, want %q", projectDir, ".omnicli/skills/")
	}
	if !contains(globalDir, "omnicli") || !contains(globalDir, "skills") {
		t.Errorf("globalDir = %q, expected to contain 'omnicli' and 'skills'", globalDir)
	}
}

func sortedEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aCopy := append([]string(nil), a...)
	bCopy := append([]string(nil), b...)
	sort.Strings(aCopy)
	sort.Strings(bCopy)
	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
