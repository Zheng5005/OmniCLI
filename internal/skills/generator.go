package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Generator collects skill information and produces a JSON file.
type Generator struct {
	Skill      Skill
	OutputPath string
}

// NewGenerator creates a generator with default output path.
func NewGenerator(name string) *Generator {
	return &Generator{
		Skill: Skill{
			Name: name,
		},
		OutputPath: filepath.Join(".omnicli", "skills", name+".json"),
	}
}

// DetectVariables extracts variable names from the system prompt.
func (g *Generator) DetectVariables() []string {
	return g.Skill.ExtractVariables()
}

// Save writes the skill to the output path as JSON.
func (g *Generator) Save() error {
	data, err := json.MarshalIndent(g.Skill, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal skill: %w", err)
	}

	dir := filepath.Dir(g.OutputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", dir, err)
	}

	if err := os.WriteFile(g.OutputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write skill file %q: %w", g.OutputPath, err)
	}

	return nil
}

// Exists returns true if the output file already exists.
func (g *Generator) Exists() bool {
	_, err := os.Stat(g.OutputPath)
	return err == nil
}
