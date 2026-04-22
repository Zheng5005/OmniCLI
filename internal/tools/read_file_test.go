package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupReadFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	var lines []string
	for i := 1; i <= 15; i++ {
		lines = append(lines, fmt.Sprintf("line %d content", i))
	}
	content := strings.Join(lines, "\n") + "\n"

	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadFile(t *testing.T) {
	tool := &ReadFileTool{}

	t.Run("full file read returns all lines with numbers", func(t *testing.T) {
		path := setupReadFile(t)
		args, _ := json.Marshal(readFileArgs{Path: path})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lines := strings.Split(result, "\n")
		if len(lines) != 15 {
			t.Errorf("expected 15 lines, got %d", len(lines))
		}

		// Check line numbers are present
		if !strings.Contains(lines[0], "1:") {
			t.Errorf("expected line 1 to have line number, got %q", lines[0])
		}
		if !strings.Contains(lines[0], "line 1 content") {
			t.Errorf("expected line 1 to have content, got %q", lines[0])
		}
		if !strings.Contains(lines[14], "15:") {
			t.Errorf("expected last line to have line number 15, got %q", lines[14])
		}
	})

	t.Run("line range read returns only requested lines", func(t *testing.T) {
		path := setupReadFile(t)
		args, _ := json.Marshal(readFileArgs{Path: path, StartLine: 3, EndLine: 5})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		lines := strings.Split(result, "\n")
		if len(lines) != 3 {
			t.Errorf("expected 3 lines, got %d", len(lines))
		}

		if !strings.Contains(lines[0], "3:") {
			t.Errorf("expected first line to be line 3, got %q", lines[0])
		}
		if !strings.Contains(lines[2], "5:") {
			t.Errorf("expected last line to be line 5, got %q", lines[2])
		}
	})

	t.Run("file not found returns message not error", func(t *testing.T) {
		fakePath := filepath.Join(t.TempDir(), "nonexistent.txt")
		args, _ := json.Marshal(readFileArgs{Path: fakePath})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := fmt.Sprintf("File not found: %s", fakePath)
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("empty path returns error", func(t *testing.T) {
		args, _ := json.Marshal(readFileArgs{Path: ""})

		_, err := tool.Execute(context.Background(), args)
		if err == nil {
			t.Error("expected error for empty path")
		}
		if !strings.Contains(err.Error(), "path is required") {
			t.Errorf("expected 'path is required' in error, got %q", err.Error())
		}
	})

	t.Run("out of range lines returns no content message", func(t *testing.T) {
		path := setupReadFile(t)
		args, _ := json.Marshal(readFileArgs{Path: path, StartLine: 100, EndLine: 200})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "No content in the specified range." {
			t.Errorf("expected 'No content in the specified range.', got %q", result)
		}
	})
}
