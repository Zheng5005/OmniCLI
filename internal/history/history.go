// Package history provides session persistence for OmniCLI conversations.
//
// Sessions are stored as JSON files in project-local (.omni/history/) or
// global (~/.config/omni/history/) directories. Writes are atomic via a
// temporary file and os.Rename to prevent data corruption.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Message represents a single chat message within a session.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Session holds the conversation state and metadata for a single session.
type Session struct {
	Messages  []Message `json:"messages"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	filePath  string
}

// ProjectHistoryDir returns the project-local history directory path.
func ProjectHistoryDir() string {
	return filepath.Join(".omni", "history")
}

// GlobalHistoryDir returns the global history directory path.
func GlobalHistoryDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "omni", "history")
	}
	return filepath.Join(homeDir, ".config", "omni", "history")
}

// NewSession creates a new session that will be stored in the given directory.
// The filename is derived from the current timestamp.
func NewSession(dir string) *Session {
	now := time.Now()
	filename := fmt.Sprintf("session_%s.json", now.Format("20060102_150405"))

	return &Session{
		Messages:  []Message{},
		CreatedAt: now,
		UpdatedAt: now,
		filePath:  filepath.Join(dir, filename),
	}
}

// AddMessage appends a message to the session and updates the timestamp.
func (s *Session) AddMessage(role, content string) {
	s.Messages = append(s.Messages, Message{
		Role:    role,
		Content: content,
	})
	s.UpdatedAt = time.Now()
}

// Save writes the session to disk atomically. It creates the parent directory
// if needed, writes to a temporary file, then renames to the final path.
func (s *Session) Save() error {
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating history directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// Load reads and unmarshals a session from the given file path.
func Load(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshaling session: %w", err)
	}

	s.filePath = path
	return &s, nil
}

// Latest returns the path of the most recent session file in the given
// directory, determined by lexicographic sort of filenames (which encodes
// the timestamp). Returns an error if no session files are found.
func Latest(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading history directory: %w", err)
	}

	var sessions []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			sessions = append(sessions, e.Name())
		}
	}

	if len(sessions) == 0 {
		return "", fmt.Errorf("no session files found in %s", dir)
	}

	sort.Strings(sessions)
	return filepath.Join(dir, sessions[len(sessions)-1]), nil
}
