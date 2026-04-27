package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// validVariableName matches allowed variable identifiers.
var validVariableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// variableRegex finds {{variable}} placeholders in text.
var variableRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)

// Skill represents a plugin/skill definition loaded from JSON.
type Skill struct {
	Name            string            `json:"name"`
	DisplayName     string            `json:"display_name"`
	SystemPrompt    string            `json:"system_prompt"`
	Variables       map[string]string `json:"variables"`
	Tools           []string          `json:"tools"`
	SafeList        []string          `json:"safe_list"`
	AutoExecuteSafe bool              `json:"auto_execute_safe"`
}

// Load reads a skill from a JSON file.
func Load(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file %q: %w", path, err)
	}

	var skill Skill
	if err := json.Unmarshal(data, &skill); err != nil {
		return nil, fmt.Errorf("failed to parse skill file %q: %w", path, err)
	}

	if err := skill.Validate(); err != nil {
		return nil, fmt.Errorf("invalid skill file %q: %w", path, err)
	}

	return &skill, nil
}

// Validate checks that required fields are present and valid.
func (s *Skill) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("required field 'name' is missing or empty")
	}

	if strings.TrimSpace(s.SystemPrompt) == "" {
		return fmt.Errorf("required field 'system_prompt' is missing or empty")
	}

	if strings.TrimSpace(s.DisplayName) == "" {
		s.DisplayName = s.Name
	}

	// Validate variable names if present.
	for name := range s.Variables {
		if !validVariableName.MatchString(name) {
			return fmt.Errorf("invalid variable name %q: must match [a-zA-Z_][a-zA-Z0-9_]*", name)
		}
	}

	// Validate variable names found in system_prompt.
	for _, name := range s.ExtractVariables() {
		if !validVariableName.MatchString(name) {
			return fmt.Errorf("invalid variable name %q in system_prompt: must match [a-zA-Z_][a-zA-Z0-9_]*", name)
		}
	}

	return nil
}

// ExtractVariables returns variable names found in the system_prompt ({{name}} syntax).
func (s *Skill) ExtractVariables() []string {
	matches := variableRegex.FindAllStringSubmatch(s.SystemPrompt, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	vars := make([]string, 0, len(matches))
	for _, m := range matches {
		name := m[1]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		vars = append(vars, name)
	}

	sort.Strings(vars)
	return vars
}

// InjectVariables replaces {{variable}} placeholders with provided values.
// Returns an error if any variable referenced in the prompt is missing from values.
func (s *Skill) InjectVariables(values map[string]string) (string, error) {
	vars := s.ExtractVariables()
	if len(vars) == 0 {
		return s.SystemPrompt, nil
	}

	// Verify all required variables are present.
	for _, name := range vars {
		if _, ok := values[name]; !ok {
			return "", fmt.Errorf("missing value for variable %q", name)
		}
	}

	result := s.SystemPrompt
	for _, name := range vars {
		placeholder := fmt.Sprintf("{{%s}}", name)
		result = strings.ReplaceAll(result, placeholder, values[name])
	}

	return result, nil
}
