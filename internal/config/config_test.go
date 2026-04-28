package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/omnicli/omnicli/internal/mcp"
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

func TestMerge_McpServers(t *testing.T) {
	dst := &Config{McpServers: map[string]mcp.ServerConfig{"a": {Type: "stdio", Command: "a"}}}
	src := &Config{McpServers: map[string]mcp.ServerConfig{"b": {Type: "sse", URL: "http://b"}}}
	merge(dst, src)
	if len(dst.McpServers) != 2 {
		t.Errorf("len = %d, want 2", len(dst.McpServers))
	}
	if _, ok := dst.McpServers["a"]; !ok {
		t.Error("expected a to remain")
	}
	if _, ok := dst.McpServers["b"]; !ok {
		t.Error("expected b to be added")
	}
}

func TestMerge_McpServersOverride(t *testing.T) {
	dst := &Config{McpServers: map[string]mcp.ServerConfig{"a": {Type: "stdio", Command: "old"}}}
	src := &Config{McpServers: map[string]mcp.ServerConfig{"a": {Type: "stdio", Command: "new"}}}
	merge(dst, src)
	if dst.McpServers["a"].Command != "new" {
		t.Errorf("command = %q, want new", dst.McpServers["a"].Command)
	}
}

func TestValidateMCPServer(t *testing.T) {
	tests := []struct {
		name    string
		srv     mcp.ServerConfig
		wantErr string
	}{
		{"stdio ok", mcp.ServerConfig{Type: "stdio", Command: "echo"}, ""},
		{"stdio missing command", mcp.ServerConfig{Type: "stdio"}, "command is required"},
		{"sse ok", mcp.ServerConfig{Type: "sse", URL: "http://x"}, ""},
		{"sse missing url", mcp.ServerConfig{Type: "sse"}, "url is required"},
		{"unknown type", mcp.ServerConfig{Type: "ws"}, "unknown transport type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMCPServer(tt.srv)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_InvalidMcpServersDropped(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	projectJSON := `{"mcp_servers":{"bad":{"type":"stdio"}}}`
	if err := os.WriteFile("omnisettings.json", []byte(projectJSON), 0o644); err != nil {
		t.Fatalf("write project config: %v", err)
	}

	cfg, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid MCP config")
	}
	if len(cfg.McpServers) != 0 {
		t.Errorf("expected invalid servers to be dropped, got %d", len(cfg.McpServers))
	}
}
