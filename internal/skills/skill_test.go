package skills

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Skill
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid skill",
			content: `{
				"name": "docs-expert",
				"display_name": "Docs Expert",
				"system_prompt": "You are a documentation expert.",
				"tools": ["read_file", "list_files"]
			}`,
			want: &Skill{
				Name:         "docs-expert",
				DisplayName:  "Docs Expert",
				SystemPrompt: "You are a documentation expert.",
				Tools:        []string{"read_file", "list_files"},
			},
			wantErr: false,
		},
		{
			name: "valid skill with variables",
			content: `{
				"name": "writer",
				"display_name": "Technical Writer",
				"system_prompt": "Write docs for {{project_name}} focusing on {{focus_area}}.",
				"variables": {
					"project_name": "Name of the project",
					"focus_area": "Documentation focus"
				},
				"tools": ["read_file"],
				"safe_list": ["^cat\\s"],
				"auto_execute_safe": true
			}`,
			want: &Skill{
				Name:            "writer",
				DisplayName:     "Technical Writer",
				SystemPrompt:    "Write docs for {{project_name}} focusing on {{focus_area}}.",
				Variables:       map[string]string{"project_name": "Name of the project", "focus_area": "Documentation focus"},
				Tools:           []string{"read_file"},
				SafeList:        []string{"^cat\\s"},
				AutoExecuteSafe: true,
			},
			wantErr: false,
		},
		{
			name:    "missing name",
			content: `{"system_prompt": "test"}`,
			wantErr: true,
			errMsg:  "name",
		},
		{
			name:    "missing system_prompt",
			content: `{"name": "test"}`,
			wantErr: true,
			errMsg:  "system_prompt",
		},
		{
			name:    "invalid json",
			content: `{"name": "test",`,
			wantErr: true,
			errMsg:  "parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "skill.json")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}

			got, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errMsg)
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/skill.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSkill_Validate(t *testing.T) {
	tests := []struct {
		name    string
		skill   Skill
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid complete skill",
			skill: Skill{
				Name:         "test",
				DisplayName:  "Test Skill",
				SystemPrompt: "You are a test.",
				Tools:        []string{"read_file"},
			},
			wantErr: false,
		},
		{
			name: "empty display_name defaults to name",
			skill: Skill{
				Name:         "my-skill",
				SystemPrompt: "prompt",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			skill: Skill{
				SystemPrompt: "prompt",
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "missing system_prompt",
			skill: Skill{
				Name: "test",
			},
			wantErr: true,
			errMsg:  "system_prompt",
		},
		{
			name: "whitespace-only name",
			skill: Skill{
				Name:         "   ",
				SystemPrompt: "prompt",
			},
			wantErr: true,
			errMsg:  "name",
		},
		{
			name: "invalid variable name in variables map",
			skill: Skill{
				Name:         "test",
				SystemPrompt: "prompt",
				Variables:    map[string]string{"123bad": "desc"},
			},
			wantErr: true,
			errMsg:  "invalid variable name",
		},
		{
			name: "invalid variable name in system_prompt",
			skill: Skill{
				Name:         "test",
				SystemPrompt: "Hello {{123bad}}",
			},
			wantErr: true,
			errMsg:  "invalid variable name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.skill.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errMsg)
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Verify display_name defaulting.
			if tt.skill.DisplayName == "" && tt.skill.Name != "" {
				if tt.skill.DisplayName != tt.skill.Name {
					// Validate mutates the receiver, so we check via a copy.
				}
			}
		})
	}
}

func TestSkill_Validate_DefaultsDisplayName(t *testing.T) {
	s := Skill{
		Name:         "my-skill",
		SystemPrompt: "prompt",
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.DisplayName != "my-skill" {
		t.Errorf("DisplayName = %q, want %q", s.DisplayName, "my-skill")
	}
}

func TestSkill_ExtractVariables(t *testing.T) {
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
		{
			name:   "mixed with regular braces",
			prompt: "JSON: {\"key\": \"value\"} and {{var}} here",
			want:   []string{"var"},
		},
		{
			name:   "empty string",
			prompt: "",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Skill{SystemPrompt: tt.prompt}
			got := s.ExtractVariables()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExtractVariables() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSkill_InjectVariables(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		values  map[string]string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "no variables",
			prompt:  "Hello world",
			values:  map[string]string{},
			want:    "Hello world",
			wantErr: false,
		},
		{
			name:    "single variable",
			prompt:  "Project: {{project_name}}",
			values:  map[string]string{"project_name": "OmniCLI"},
			want:    "Project: OmniCLI",
			wantErr: false,
		},
		{
			name:    "multiple variables",
			prompt:  "{{greeting}} {{name}}!",
			values:  map[string]string{"greeting": "Hello", "name": "World"},
			want:    "Hello World!",
			wantErr: false,
		},
		{
			name:    "missing variable",
			prompt:  "{{name}}",
			values:  map[string]string{},
			wantErr: true,
			errMsg:  "missing value for variable",
		},
		{
			name:    "partial missing variable",
			prompt:  "{{a}} and {{b}}",
			values:  map[string]string{"a": "A"},
			wantErr: true,
			errMsg:  "missing value for variable \"b\"",
		},
		{
			name:    "duplicate variable replacement",
			prompt:  "{{x}} {{x}}",
			values:  map[string]string{"x": "hi"},
			want:    "hi hi",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Skill{SystemPrompt: tt.prompt}
			got, err := s.InjectVariables(tt.values)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errMsg)
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("InjectVariables() = %q, want %q", got, tt.want)
			}
		})
	}
}


