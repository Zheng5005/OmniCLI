package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string // empty means don't create the file
		wantOK   bool
		wantCfg  *Config
	}{
		{
			name:    "valid JSON file",
			content: `{"modelPriority":["claude-3"],"theme":{"primaryColor":"#ff0000"}}`,
			wantOK:  true,
			wantCfg: &Config{
				ModelPriority: []string{"claude-3"},
				Theme:         ThemeConfig{PrimaryColor: "#ff0000"},
			},
		},
		{
			name:    "missing file",
			content: "",
			wantOK:  false,
			wantCfg: nil,
		},
		{
			name:    "invalid JSON",
			content: `{not valid json`,
			wantOK:  false,
			wantCfg: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "omnisettings.json")

			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatalf("writing test file: %v", err)
				}
			}

			cfg, ok := loadFile(path)

			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}

			if !tt.wantOK {
				if cfg != nil {
					t.Errorf("cfg = %+v, want nil", cfg)
				}
				return
			}

			if !reflect.DeepEqual(cfg, tt.wantCfg) {
				t.Errorf("cfg = %+v, want %+v", cfg, tt.wantCfg)
			}
		})
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		dst  *Config
		src  *Config
		want *Config
	}{
		{
			name: "ModelPriority overrides dest",
			dst:  &Config{ModelPriority: []string{"gpt-4o"}},
			src:  &Config{ModelPriority: []string{"claude-3"}},
			want: &Config{ModelPriority: []string{"claude-3"}},
		},
		{
			name: "empty ModelPriority does not override",
			dst:  &Config{ModelPriority: []string{"gpt-4o"}},
			src:  &Config{ModelPriority: nil},
			want: &Config{ModelPriority: []string{"gpt-4o"}},
		},
		{
			name: "Theme overrides dest",
			dst:  &Config{Theme: ThemeConfig{PrimaryColor: "#000"}},
			src:  &Config{Theme: ThemeConfig{PrimaryColor: "#fff"}},
			want: &Config{Theme: ThemeConfig{PrimaryColor: "#fff"}},
		},
		{
			name: "empty Theme does not override",
			dst:  &Config{Theme: ThemeConfig{PrimaryColor: "#000"}},
			src:  &Config{Theme: ThemeConfig{}},
			want: &Config{Theme: ThemeConfig{PrimaryColor: "#000"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merge(tt.dst, tt.src)

			if !reflect.DeepEqual(tt.dst, tt.want) {
				t.Errorf("after merge: got %+v, want %+v", tt.dst, tt.want)
			}
		})
	}
}

func TestDefaults(t *testing.T) {
	cfg := defaults()

	if !reflect.DeepEqual(cfg.ModelPriority, []string{"gpt-4o"}) {
		t.Errorf("ModelPriority = %v, want [gpt-4o]", cfg.ModelPriority)
	}

	if len(cfg.AllowedCommands) != 0 {
		t.Errorf("AllowedCommands = %v, want empty", cfg.AllowedCommands)
	}

	if (cfg.Theme != ThemeConfig{}) {
		t.Errorf("Theme = %+v, want zero value", cfg.Theme)
	}
}
