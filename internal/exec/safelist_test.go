package exec

import (
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	compiled, errs := CompilePatterns(DefaultPatterns())
	if len(errs) > 0 {
		t.Fatalf("default patterns should compile without errors: %v", errs)
	}

	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{"go test matches", "go test ./...", true},
		{"go fmt matches", "go fmt ./cmd/", true},
		{"go build matches", "go build -o bin", true},
		{"git status matches", "git status", true},
		{"git diff matches", "git diff HEAD", true},
		{"rm -rf blocked", "rm -rf /", false},
		{"curl blocked", "curl https://evil.com", false},
		{"docker blocked", "docker run nginx", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Match(tt.cmd, compiled)
			if got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestCompilePatterns(t *testing.T) {
	t.Run("invalid regex returns errors", func(t *testing.T) {
		patterns := []string{
			`^go\s+test\b`, // valid
			`[invalid`,     // invalid
			`^git\s+log\b`, // valid
			`(unclosed`,    // invalid
		}

		compiled, errs := CompilePatterns(patterns)

		if len(compiled) != 2 {
			t.Errorf("expected 2 compiled patterns, got %d", len(compiled))
		}
		if len(errs) != 2 {
			t.Errorf("expected 2 errors, got %d", len(errs))
		}

		for _, e := range errs {
			if !strings.Contains(e.Error(), "invalid safe-list pattern") {
				t.Errorf("expected error to mention 'invalid safe-list pattern', got %q", e.Error())
			}
		}
	})
}

func TestDefaultPatterns(t *testing.T) {
	patterns := DefaultPatterns()
	if len(patterns) != 10 {
		t.Errorf("expected 10 default patterns, got %d", len(patterns))
	}
}
