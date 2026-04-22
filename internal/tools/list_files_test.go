package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupListFilesDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create nested structure
	dirs := []string{
		"src",
		"src/pkg",
		"src/pkg/deep",
		"build",
		".git",
		".git/objects",
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	files := map[string]string{
		"main.go":           "package main",
		"README.md":         "# Project",
		"src/lib.go":        "package src",
		"src/pkg/util.go":   "package pkg",
		"src/pkg/deep/a.go": "package deep",
		"build/app.log":     "log line",
		"build/output.log":  "another log",
		".git/HEAD":         "ref: refs/heads/main",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func TestListFiles(t *testing.T) {
	tool := &ListFilesTool{}

	t.Run("tree format output", func(t *testing.T) {
		dir := setupListFilesDir(t)
		args, _ := json.Marshal(listFilesArgs{Path: dir, MaxDepth: 10})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should start with root dir name
		lines := strings.Split(result, "\n")
		if !strings.HasSuffix(lines[0], "/") {
			t.Errorf("first line should end with /, got %q", lines[0])
		}

		// Should contain known files
		if !strings.Contains(result, "main.go") {
			t.Error("expected main.go in output")
		}
		if !strings.Contains(result, "src/") {
			t.Error("expected src/ directory in output")
		}
	})

	t.Run("omniignore filters", func(t *testing.T) {
		dir := setupListFilesDir(t)
		// Create .omniignore that excludes *.log
		if err := os.WriteFile(filepath.Join(dir, ".omniignore"), []byte("*.log\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		args, _ := json.Marshal(listFilesArgs{Path: dir, MaxDepth: 10})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, "app.log") {
			t.Error("expected *.log files to be excluded")
		}
		if strings.Contains(result, "output.log") {
			t.Error("expected *.log files to be excluded")
		}
		if !strings.Contains(result, "main.go") {
			t.Error("expected main.go to still be present")
		}
	})

	t.Run("git always excluded", func(t *testing.T) {
		dir := setupListFilesDir(t)
		// No .omniignore — .git should still be excluded
		args, _ := json.Marshal(listFilesArgs{Path: dir, MaxDepth: 10})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if strings.Contains(result, ".git") {
			t.Error("expected .git to be excluded even without .omniignore")
		}
	})

	t.Run("max_depth limits traversal", func(t *testing.T) {
		dir := setupListFilesDir(t)
		args, _ := json.Marshal(listFilesArgs{Path: dir, MaxDepth: 1})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Depth 1: should see src/ but not src/pkg/
		if !strings.Contains(result, "src/") {
			t.Error("expected src/ at depth 1")
		}
		if strings.Contains(result, "util.go") {
			t.Error("expected util.go (depth 3) to be excluded at max_depth=1")
		}
		if strings.Contains(result, "deep/") {
			t.Error("expected deep/ (depth 3) to be excluded at max_depth=1")
		}
	})

	t.Run("missing path returns only root line", func(t *testing.T) {
		args, _ := json.Marshal(listFilesArgs{Path: "/nonexistent/path/xyz"})

		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// WalkDir on a missing path produces an error at the root which is
		// swallowed, so we only get the root line with no children.
		lines := strings.Split(result, "\n")
		if len(lines) != 1 {
			t.Errorf("expected 1 line (root only) for missing path, got %d: %v", len(lines), lines)
		}
	})
}
