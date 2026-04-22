package exec

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClassify(t *testing.T) {
	compiled, _ := CompilePatterns(DefaultPatterns())

	tests := []struct {
		name string
		cmd  string
		want Classification
	}{
		{"safe command", "go test ./...", Safe},
		{"risky command", "rm -rf /", Risky},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.cmd, compiled)
			if got != tt.want {
				t.Errorf("Classify(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestClassificationString(t *testing.T) {
	tests := []struct {
		name string
		c    Classification
		want string
	}{
		{"safe string", Safe, "safe"},
		{"risky string", Risky, "risky"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("Classification.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRun(t *testing.T) {
	t.Run("echo hello", func(t *testing.T) {
		result, err := Run(context.Background(), "echo hello", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Stdout != "hello\n" {
			t.Errorf("stdout = %q, want %q", result.Stdout, "hello\n")
		}
		if result.ExitCode != 0 {
			t.Errorf("exit code = %d, want 0", result.ExitCode)
		}
	})

	t.Run("exit 1 returns non-zero exit code", func(t *testing.T) {
		result, err := Run(context.Background(), "exit 1", 0)
		if err != nil {
			t.Fatalf("non-zero exit should not be an error, got: %v", err)
		}
		if result.ExitCode != 1 {
			t.Errorf("exit code = %d, want 1", result.ExitCode)
		}
	})

	t.Run("nonexistent command returns exit code 127", func(t *testing.T) {
		// sh -c wraps the command, so a missing command yields exit 127
		result, err := Run(context.Background(), "nonexistent_cmd_abc123", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 127 {
			t.Errorf("exit code = %d, want 127", result.ExitCode)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		_, err := Run(context.Background(), "sleep 10", 200*time.Millisecond)
		if err == nil {
			t.Fatal("expected timeout error")
		}
		if !strings.Contains(err.Error(), "timed out") {
			t.Errorf("expected timeout error, got: %v", err)
		}
	})
}
