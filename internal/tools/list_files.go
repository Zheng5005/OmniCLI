package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// ListFilesTool implements the list_files tool that generates a project
// directory tree, respecting .omniignore patterns.
type ListFilesTool struct{}

type listFilesArgs struct {
	Path     string `json:"path"`
	MaxDepth int    `json:"max_depth"`
}

// Name returns the tool name.
func (t *ListFilesTool) Name() string { return "list_files" }

// Description returns what the tool does.
func (t *ListFilesTool) Description() string {
	return "Lists files and directories in a tree-like format, respecting .omniignore patterns."
}

// Parameters returns the JSON Schema for the tool's arguments.
func (t *ListFilesTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "path": {
      "type": "string",
      "description": "Root path to list from. Defaults to current directory.",
      "default": "."
    },
    "max_depth": {
      "type": "integer",
      "description": "Maximum depth to traverse. -1 for unlimited.",
      "default": 3
    }
  }
}`)
}

// Execute runs the list_files tool.
func (t *ListFilesTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a listFilesArgs
	a.Path = "."
	a.MaxDepth = 3

	if len(args) > 0 {
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		// Re-apply defaults for zero values that weren't explicitly set
		if a.Path == "" {
			a.Path = "."
		}
	}

	absRoot, err := filepath.Abs(a.Path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	shouldIgnore := LoadIgnorePatterns(absRoot)

	var lines []string
	lines = append(lines, filepath.Base(absRoot)+"/")

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip entries we can't read
		}

		// Skip root itself
		if path == absRoot {
			return nil
		}

		rel, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			return nil
		}

		depth := strings.Count(rel, string(filepath.Separator)) + 1

		if shouldIgnore(rel, d.IsDir()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if a.MaxDepth >= 0 && depth > a.MaxDepth {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		indent := strings.Repeat("  ", depth)
		name := d.Name()
		if d.IsDir() {
			name += "/"
		}
		lines = append(lines, indent+name)

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error walking directory: %w", err)
	}

	return strings.Join(lines, "\n"), nil
}
