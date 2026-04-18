package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ReadFileTool implements the read_file tool that reads file content
// with optional line range selection.
type ReadFileTool struct{}

type readFileArgs struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// Name returns the tool name.
func (t *ReadFileTool) Name() string { return "read_file" }

// Description returns what the tool does.
func (t *ReadFileTool) Description() string {
	return "Reads file content with optional line range (1-indexed)."
}

// Parameters returns the JSON Schema for the tool's arguments.
func (t *ReadFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {
      "type": "string",
      "description": "Path to the file to read."
    },
    "start_line": {
      "type": "integer",
      "description": "Start line number (1-indexed, inclusive)."
    },
    "end_line": {
      "type": "integer",
      "description": "End line number (1-indexed, inclusive)."
    }
  }
}`)
}

// Execute runs the read_file tool.
func (t *ReadFileTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a readFileArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if a.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	f, err := os.Open(a.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("File not found: %s", a.Path), nil
		}
		return "", fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++

		if a.StartLine > 0 && lineNum < a.StartLine {
			continue
		}
		if a.EndLine > 0 && lineNum > a.EndLine {
			break
		}

		lines = append(lines, fmt.Sprintf("%4d: %s", lineNum, scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	if len(lines) == 0 {
		return "No content in the specified range.", nil
	}

	return strings.Join(lines, "\n"), nil
}
