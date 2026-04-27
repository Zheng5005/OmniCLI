package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Manager handles loading and listing skills from project and global directories.
type Manager struct {
	projectDir string // .omnicli/skills/
	globalDir  string // ~/.config/omnicli/skills/
	cache      map[string]*Skill
}

// NewManager creates a Manager. projectDir can be empty (no project skills).
func NewManager(projectDir string) *Manager {
	return &Manager{
		projectDir: projectDir,
		cache:      make(map[string]*Skill),
	}
}

// SetGlobalDir sets the global skill directory.
func (m *Manager) SetGlobalDir(dir string) {
	m.globalDir = dir
}

// DefaultDirs returns the default project and global skill directories.
func DefaultDirs() (projectDir, globalDir string) {
	projectDir = ".omnicli/skills/"

	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	globalDir = filepath.Join(configDir, "omnicli", "skills")

	return projectDir, globalDir
}

// Get returns a skill by name, checking project dir first, then global.
func (m *Manager) Get(name string) (*Skill, error) {
	// Check cache first.
	if skill, ok := m.cache[name]; ok {
		return skill, nil
	}

	// Search project dir first.
	if m.projectDir != "" {
		path := filepath.Join(m.projectDir, name+".json")
		if _, err := os.Stat(path); err == nil {
			skill, err := Load(path)
			if err != nil {
				return nil, err
			}
			m.cache[name] = skill
			return skill, nil
		}
	}

	// Fallback to global dir.
	if m.globalDir != "" {
		path := filepath.Join(m.globalDir, name+".json")
		if _, err := os.Stat(path); err == nil {
			skill, err := Load(path)
			if err != nil {
				return nil, err
			}
			m.cache[name] = skill
			return skill, nil
		}
	}

	return nil, fmt.Errorf("skill %q not found in %s", name, m.searchPaths())
}

// List returns all available skill names (project + global, deduplicated).
func (m *Manager) List() ([]string, error) {
	seen := make(map[string]struct{})
	names := make([]string, 0)

	// Scan project dir first so project skills take precedence
	// (they're added first and global duplicates are skipped).
	if m.projectDir != "" {
		projectNames, _ := listJSONFiles(m.projectDir)
		for _, name := range projectNames {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}

	// Scan global dir.
	if m.globalDir != "" {
		globalNames, _ := listJSONFiles(m.globalDir)
		for _, name := range globalNames {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names, nil
}

// searchPaths returns a human-readable list of directories searched.
func (m *Manager) searchPaths() string {
	paths := make([]string, 0, 2)
	if m.projectDir != "" {
		paths = append(paths, m.projectDir)
	}
	if m.globalDir != "" {
		paths = append(paths, m.globalDir)
	}
	return strings.Join(paths, " or ")
}

// listJSONFiles returns the basenames (without .json extension) of all
// .json files in dir. If the directory doesn't exist or can't be read,
// it returns an empty slice (no error) — missing skill dirs are normal.
func listJSONFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Missing or unreadable directories are normal — treat as empty.
		return nil, nil
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".json") {
			names = append(names, strings.TrimSuffix(name, ".json"))
		}
	}

	sort.Strings(names)
	return names, nil
}
