package security

import (
	"bytes"
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantSubstr string // must appear in output
		noSubstr   string // must NOT appear in output (empty = skip check)
	}{
		{
			name:       "OpenAI key",
			input:      "key is sk-abc123def456ghi789jkl012",
			wantSubstr: "[REDACTED]",
			noSubstr:   "sk-",
		},
		{
			name:       "Google AI key",
			input:      "AIzaSyBcDeFgHiJkLmNoPqRsTuVwXyZ0123456789a",
			wantSubstr: "[REDACTED]",
			noSubstr:   "AIza",
		},
		{
			name:       "GitHub PAT",
			input:      "ghp_abcdefghijklmnopqrstuvwxyz0123456789",
			wantSubstr: "[REDACTED]",
			noSubstr:   "ghp_",
		},
		{
			name:       "GitHub OAuth",
			input:      "gho_abcdefghijklmnopqrstuvwxyz0123456789",
			wantSubstr: "[REDACTED]",
			noSubstr:   "gho_",
		},
		{
			name:       "env var with equals",
			input:      "API_KEY=mysecretvalue",
			wantSubstr: "[REDACTED]",
			noSubstr:   "mysecretvalue",
		},
		{
			name:       "env var with colon",
			input:      "token: supersecret123",
			wantSubstr: "[REDACTED]",
			noSubstr:   "supersecret123",
		},
		{
			name:       "normal text unchanged",
			input:      "Hello world, this is normal",
			wantSubstr: "Hello world, this is normal",
		},
		{
			name:       "mixed content preserves non-sensitive",
			input:      "My key is sk-aaaabbbbccccddddeeeefffff and name is John",
			wantSubstr: "John",
			noSubstr:   "sk-aaaa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Redact(tt.input)

			if !strings.Contains(result, tt.wantSubstr) {
				t.Errorf("result %q does not contain %q", result, tt.wantSubstr)
			}

			if tt.noSubstr != "" && strings.Contains(result, tt.noSubstr) {
				t.Errorf("result %q should not contain %q", result, tt.noSubstr)
			}
		})
	}
}

func TestRedactingWriter(t *testing.T) {
	t.Run("redacts output", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewRedactingWriter(&buf)

		input := []byte("my key is sk-abc123def456ghi789jkl012")
		n, err := w.Write(input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if n != len(input) {
			t.Errorf("byte count = %d, want %d", n, len(input))
		}

		output := buf.String()
		if strings.Contains(output, "sk-") {
			t.Errorf("output %q still contains sensitive data", output)
		}
		if !strings.Contains(output, "[REDACTED]") {
			t.Errorf("output %q missing [REDACTED]", output)
		}
	})

	t.Run("extra patterns work", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewRedactingWriter(&buf, `CUSTOM-\d+`)

		input := []byte("value is CUSTOM-999")
		_, err := w.Write(input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		output := buf.String()
		if strings.Contains(output, "CUSTOM-999") {
			t.Errorf("output %q should have redacted custom pattern", output)
		}
		if !strings.Contains(output, "[REDACTED]") {
			t.Errorf("output %q missing [REDACTED]", output)
		}
	})

	t.Run("invalid extra pattern silently skipped", func(t *testing.T) {
		var buf bytes.Buffer
		w := NewRedactingWriter(&buf, `[invalid`)

		if w == nil {
			t.Fatal("writer should not be nil even with invalid pattern")
		}

		input := []byte("normal text")
		n, err := w.Write(input)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n != len(input) {
			t.Errorf("byte count = %d, want %d", n, len(input))
		}
		if buf.String() != "normal text" {
			t.Errorf("output = %q, want %q", buf.String(), "normal text")
		}
	})
}
