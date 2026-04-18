package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const maxGrepMatches = 100

// GrepSearchTool implements the grep_search tool that searches file
// contents using regular expressions.
type GrepSearchTool struct{}

type grepSearchArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
	Include string `json:"include"`
}

// Name returns the tool name.
func (t *GrepSearchTool) Name() string { return "grep_search" }

// Description returns what the tool does.
func (t *GrepSearchTool) Description() string {
	return "Searches file contents using regex patterns, respecting .omniignore."
}

// Parameters returns the JSON Schema for the tool's arguments.
func (t *GrepSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "required": ["pattern"],
  "properties": {
    "pattern": {
      "type": "string",
      "description": "Regex pattern to search for."
    },
    "path": {
      "type": "string",
      "description": "Root path to search from. Defaults to current directory.",
      "default": "."
    },
    "include": {
      "type": "string",
      "description": "File glob filter, e.g. *.go"
    }
  }
}`)
}

// Execute runs the grep_search tool.
func (t *GrepSearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a grepSearchArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &a); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}

	if a.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}
	if a.Path == "" {
		a.Path = "."
	}

	re, err := regexp.Compile(a.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	absRoot, err := filepath.Abs(a.Path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	shouldIgnore := LoadIgnorePatterns(absRoot)

	var matches []string

	walkErr := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if path == absRoot {
			return nil
		}

		rel, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			return nil
		}

		if shouldIgnore(rel, d.IsDir()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			return nil
		}

		// Apply include filter
		if a.Include != "" {
			matched, _ := filepath.Match(a.Include, d.Name())
			if !matched {
				return nil
			}
		}

		// Skip binary files
		if isBinary(path) {
			return nil
		}

		fileMatches, scanErr := searchFile(path, rel, re)
		if scanErr != nil {
			return nil
		}

		matches = append(matches, fileMatches...)
		if len(matches) >= maxGrepMatches {
			return fs.SkipAll
		}

		return nil
	})
	if walkErr != nil {
		return "", fmt.Errorf("error walking directory: %w", walkErr)
	}

	if len(matches) == 0 {
		return "No matches found", nil
	}

	if len(matches) > maxGrepMatches {
		matches = matches[:maxGrepMatches]
	}

	result := strings.Join(matches, "\n")
	if len(matches) == maxGrepMatches {
		result += fmt.Sprintf("\n\n(results truncated at %d matches)", maxGrepMatches)
	}

	return result, nil
}

func searchFile(absPath, relPath string, re *regexp.Regexp) ([]string, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var matches []string
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			matches = append(matches, fmt.Sprintf("%s:%d: %s", relPath, lineNum, line))
		}
	}
	return matches, scanner.Err()
}

func isBinary(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil {
		return false
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}
