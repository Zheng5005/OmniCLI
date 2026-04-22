package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupGrepDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"main.go": `package main

func main() {
	fmt.Println("hello world")
}
`,
		"lib.go": `package main

func TestHelper() string {
	return "test"
}
`,
		"readme.txt": `This is a readme file.
It contains hello in it.
Nothing else here.
`,
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func TestGrepSearch(t *testing.T) {
	tool := &GrepSearchTool{}

	t.Run("keyword search finds matches", func(t *testing.T) {
		dir := setupGrepDir(t)
		args, _ := json.Marshal(grepSearchArgs{Pattern: "hello", Path: dir})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "hello") {
			t.Error("expected result to contain 'hello'")
		}
		// Should have file:line:content format
		for _, line := range strings.Split(result, "\n") {
			parts := strings.SplitN(line, ":", 3)
			if len(parts) < 3 {
				t.Errorf("expected file:line:content format, got %q", line)
			}
		}
	})

	t.Run("regex search works", func(t *testing.T) {
		dir := setupGrepDir(t)
		args, _ := json.Marshal(grepSearchArgs{Pattern: `func\s+Test`, Path: dir})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "TestHelper") {
			t.Error("expected regex to match func TestHelper")
		}
		// Should not match non-Test funcs
		if strings.Contains(result, "func main") {
			t.Error("expected regex not to match func main")
		}
	})

	t.Run("no matches returns message", func(t *testing.T) {
		dir := setupGrepDir(t)
		args, _ := json.Marshal(grepSearchArgs{Pattern: "zzz_nonexistent_zzz", Path: dir})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "No matches found" {
			t.Errorf("expected 'No matches found', got %q", result)
		}
	})

	t.Run("include filter works", func(t *testing.T) {
		dir := setupGrepDir(t)
		args, _ := json.Marshal(grepSearchArgs{Pattern: "hello", Path: dir, Include: "*.go"})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(result, "main.go") {
			t.Error("expected match in main.go")
		}
		if strings.Contains(result, "readme.txt") {
			t.Error("expected readme.txt to be excluded by *.go filter")
		}
	})

	t.Run("invalid regex returns error", func(t *testing.T) {
		dir := setupGrepDir(t)
		args, _ := json.Marshal(grepSearchArgs{Pattern: "[invalid", Path: dir})

		_, err := tool.Execute(context.Background(), args)
		if err == nil {
			t.Error("expected error for invalid regex")
		}
		if !strings.Contains(err.Error(), "invalid regex") {
			t.Errorf("expected 'invalid regex' in error, got %q", err.Error())
		}
	})

	t.Run("binary files are skipped", func(t *testing.T) {
		dir := setupGrepDir(t)

		// Create a binary file with a null byte
		binaryContent := []byte("hello\x00world binary")
		if err := os.WriteFile(filepath.Join(dir, "data.bin"), binaryContent, 0o644); err != nil {
			t.Fatal(err)
		}

		args, _ := json.Marshal(grepSearchArgs{Pattern: "hello", Path: dir})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "data.bin") {
			t.Error("expected binary file to be skipped")
		}
	})
}
