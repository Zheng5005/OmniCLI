// Package config provides configuration loading and merging for OmniCLI.
//
// Configuration is resolved from a hierarchy: project-level omnisettings.json
// overrides global ~/.config/omni/omnisettings.json, with built-in defaults
// as the final fallback. Top-level keys are shallow-merged, except
// AllowedCommands which concatenates project and global arrays.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ThemeConfig holds terminal color theming preferences.
type ThemeConfig struct {
	PrimaryColor   string `json:"primaryColor"`
	SecondaryColor string `json:"secondaryColor"`
	AccentColor    string `json:"accentColor"`
}

// Config holds the full application configuration.
type Config struct {
	ModelPriority   []string    `json:"modelPriority"`
	AllowedCommands []string    `json:"allowedCommands"`
	Theme           ThemeConfig `json:"theme"`
}

// defaults returns the built-in default configuration.
func defaults() *Config {
	return &Config{
		ModelPriority:   []string{"gpt-4o"},
		AllowedCommands: []string{},
		Theme:           ThemeConfig{},
	}
}

// Load resolves configuration from the hierarchy:
//  1. Project-level: omnisettings.json in the current directory
//  2. Global: ~/.config/omni/omnisettings.json
//  3. Built-in defaults
//
// Missing config files are not errors — defaults are used silently.
// Top-level keys from the project config override global config (shallow merge),
// except AllowedCommands which concatenates both arrays (project + global).
func Load() (*Config, error) {
	base := defaults()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return base, nil
	}

	globalPath := filepath.Join(homeDir, ".config", "omni", "omnisettings.json")
	globalCfg, globalOK := loadFile(globalPath)

	projectPath := "omnisettings.json"
	projectCfg, projectOK := loadFile(projectPath)

	if !globalOK && !projectOK {
		return base, nil
	}

	if globalOK {
		merge(base, globalCfg)
	}

	if projectOK {
		// Save global AllowedCommands before project merge overwrites them.
		globalCommands := make([]string, len(base.AllowedCommands))
		copy(globalCommands, base.AllowedCommands)

		merge(base, projectCfg)

		// Concatenate: project commands come after global commands.
		if len(projectCfg.AllowedCommands) > 0 {
			base.AllowedCommands = append(globalCommands, projectCfg.AllowedCommands...)
		}
	}

	return base, nil
}

// loadFile reads and unmarshals a JSON config file. It returns the parsed
// config and true on success, or nil and false if the file cannot be read.
func loadFile(path string) (*Config, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, false
	}

	return &cfg, true
}

// merge performs a shallow merge of src into dst. Only non-zero top-level
// fields in src overwrite the corresponding field in dst.
func merge(dst, src *Config) {
	if len(src.ModelPriority) > 0 {
		dst.ModelPriority = src.ModelPriority
	}
	if len(src.AllowedCommands) > 0 {
		dst.AllowedCommands = src.AllowedCommands
	}
	if (src.Theme != ThemeConfig{}) {
		dst.Theme = src.Theme
	}
}
