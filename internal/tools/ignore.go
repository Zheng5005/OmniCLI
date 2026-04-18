package tools

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// IgnoreFunc returns true if the given path should be ignored.
type IgnoreFunc func(path string, isDir bool) bool

// LoadIgnorePatterns reads an .omniignore file from rootPath and returns
// an IgnoreFunc that checks whether a path should be ignored.
// The .git directory is always ignored regardless of .omniignore content.
// If the .omniignore file does not exist, only .git is ignored.
func LoadIgnorePatterns(rootPath string) IgnoreFunc {
	patterns := parseIgnoreFile(filepath.Join(rootPath, ".omniignore"))

	return func(path string, isDir bool) bool {
		// Always ignore .git
		name := filepath.Base(path)
		if name == ".git" && isDir {
			return true
		}

		for _, p := range patterns {
			if p.dirOnly && !isDir {
				continue
			}
			if matched, _ := filepath.Match(p.pattern, name); matched {
				return true
			}
		}
		return false
	}
}

type ignorePattern struct {
	pattern string
	dirOnly bool
}

func parseIgnoreFile(path string) []ignorePattern {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var patterns []ignorePattern
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		p := ignorePattern{}
		if strings.HasSuffix(line, "/") {
			p.dirOnly = true
			p.pattern = strings.TrimSuffix(line, "/")
		} else {
			p.pattern = line
		}
		patterns = append(patterns, p)
	}
	return patterns
}
