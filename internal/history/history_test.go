package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	dir := t.TempDir()
	before := time.Now()
	s := NewSession(dir)
	after := time.Now()

	if len(s.Messages) != 0 {
		t.Errorf("Messages = %v, want empty", s.Messages)
	}

	if s.CreatedAt.Before(before) || s.CreatedAt.After(after) {
		t.Errorf("CreatedAt = %v, want between %v and %v", s.CreatedAt, before, after)
	}

	if !strings.HasPrefix(filepath.Base(s.filePath), "session_") {
		t.Errorf("filePath base = %q, want prefix session_", filepath.Base(s.filePath))
	}

	if !strings.HasSuffix(s.filePath, ".json") {
		t.Errorf("filePath = %q, want .json suffix", s.filePath)
	}

	if filepath.Dir(s.filePath) != dir {
		t.Errorf("filePath dir = %q, want %q", filepath.Dir(s.filePath), dir)
	}
}

func TestAddMessage(t *testing.T) {
	s := NewSession(t.TempDir())
	beforeAdd := time.Now()

	s.AddMessage("user", "hello")
	s.AddMessage("assistant", "hi there")

	if len(s.Messages) != 2 {
		t.Fatalf("Messages count = %d, want 2", len(s.Messages))
	}

	if s.Messages[0].Role != "user" || s.Messages[0].Content != "hello" {
		t.Errorf("Messages[0] = %+v, want {user, hello}", s.Messages[0])
	}

	if s.Messages[1].Role != "assistant" || s.Messages[1].Content != "hi there" {
		t.Errorf("Messages[1] = %+v, want {assistant, hi there}", s.Messages[1])
	}

	if s.UpdatedAt.Before(beforeAdd) {
		t.Errorf("UpdatedAt = %v, should be after %v", s.UpdatedAt, beforeAdd)
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	s := NewSession(dir)
	s.Model = "gpt-4o"
	s.AddMessage("user", "test question")
	s.AddMessage("assistant", "test answer")

	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := Load(s.filePath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.Model != s.Model {
		t.Errorf("Model = %q, want %q", loaded.Model, s.Model)
	}

	if len(loaded.Messages) != len(s.Messages) {
		t.Fatalf("Messages count = %d, want %d", len(loaded.Messages), len(s.Messages))
	}

	for i, msg := range loaded.Messages {
		if msg.Role != s.Messages[i].Role || msg.Content != s.Messages[i].Content {
			t.Errorf("Messages[%d] = %+v, want %+v", i, msg, s.Messages[i])
		}
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	s := NewSession(dir)
	s.AddMessage("user", "hi")

	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("directory %q was not created", dir)
	}
}

func TestSaveAtomicNoTmpFile(t *testing.T) {
	dir := t.TempDir()
	s := NewSession(dir)
	s.AddMessage("user", "hi")

	if err := s.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir error: %v", err)
	}

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("found leftover temp file: %s", e.Name())
		}
	}
}

func TestLatest(t *testing.T) {
	t.Run("returns most recent by name sort", func(t *testing.T) {
		dir := t.TempDir()

		files := []string{
			"session_20240101_100000.json",
			"session_20240103_100000.json",
			"session_20240102_100000.json",
		}
		for _, f := range files {
			if err := os.WriteFile(filepath.Join(dir, f), []byte("{}"), 0o644); err != nil {
				t.Fatalf("writing file: %v", err)
			}
		}

		got, err := Latest(dir)
		if err != nil {
			t.Fatalf("Latest() error: %v", err)
		}

		want := filepath.Join(dir, "session_20240103_100000.json")
		if got != want {
			t.Errorf("Latest() = %q, want %q", got, want)
		}
	})

	t.Run("empty directory returns error", func(t *testing.T) {
		dir := t.TempDir()

		_, err := Latest(dir)
		if err == nil {
			t.Error("Latest() should return error for empty directory")
		}
	})

	t.Run("no JSON files returns error", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hi"), 0o644); err != nil {
			t.Fatalf("writing file: %v", err)
		}

		_, err := Latest(dir)
		if err == nil {
			t.Error("Latest() should return error when no JSON files exist")
		}
	})
}

func TestProjectHistoryDir(t *testing.T) {
	got := ProjectHistoryDir()
	want := filepath.Join(".omni", "history")

	if got != want {
		t.Errorf("ProjectHistoryDir() = %q, want %q", got, want)
	}
}

func TestGlobalHistoryDir(t *testing.T) {
	got := GlobalHistoryDir()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home dir: %v", err)
	}

	want := filepath.Join(homeDir, ".config", "omni", "history")
	if got != want {
		t.Errorf("GlobalHistoryDir() = %q, want %q", got, want)
	}
}
