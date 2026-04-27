package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/omnicli/omnicli/internal/skills"
)

// validToolNames are the known tool names that can be selected.
var validToolNames = []string{"list_files", "grep_search", "read_file", "run_command"}

func runSkillInit(args []string) {
	if len(args) == 0 || args[0] != "init" {
		fmt.Fprintln(os.Stderr, "Usage: omni skill init [--output path]")
		os.Exit(1)
	}

	initCmd := flag.NewFlagSet("init", flag.ExitOnError)
	outputPath := initCmd.String("output", "", "Output path for the skill JSON file")
	if err := initCmd.Parse(args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)

	// 1. Skill name
	name := promptRequired(reader, "Skill name")
	if strings.TrimSpace(name) == "" {
		fmt.Fprintln(os.Stderr, "Skill name is required.")
		os.Exit(1)
	}

	// 2. Display name
	displayName := prompt(reader, "Display name (optional, defaults to name)")

	// 3. System prompt
	systemPrompt := promptRequired(reader, "System prompt")

	// Create generator
	gen := skills.NewGenerator(name)
	if *outputPath != "" {
		gen.OutputPath = *outputPath
	}
	gen.Skill.DisplayName = displayName
	gen.Skill.SystemPrompt = systemPrompt

	// 4. Tool selection
	fmt.Println("Available tools:")
	for i, t := range validToolNames {
		fmt.Printf("  %d. %s\n", i+1, t)
	}
	toolsInput := prompt(reader, "Select tools (comma-separated names or numbers)")
	selectedTools := parseToolSelection(toolsInput)
	gen.Skill.Tools = selectedTools

	// 5. Safe list commands
	safeListInput := prompt(reader, "Safe list commands (comma-separated, e.g., \"ls, git status\")")
	if safeListInput != "" {
		gen.Skill.SafeList = splitAndTrim(safeListInput)
	}

	// 6. Auto-execute safe commands
	autoExec := prompt(reader, "Auto-execute safe commands? (y/n)")
	gen.Skill.AutoExecuteSafe = strings.HasPrefix(strings.ToLower(strings.TrimSpace(autoExec)), "y")

	// 7. Detect variables and prompt for descriptions
	vars := gen.DetectVariables()
	if len(vars) > 0 {
		fmt.Printf("Detected %d variable(s) in system prompt.\n", len(vars))
		gen.Skill.Variables = make(map[string]string)
		for _, v := range vars {
			desc := prompt(reader, fmt.Sprintf("Description for '%s'", v))
			gen.Skill.Variables[v] = desc
		}
	}

	// Check for overwrite
	if gen.Exists() {
		confirm := prompt(reader, fmt.Sprintf("File %s already exists. Overwrite? (y/n)", gen.OutputPath))
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(confirm)), "y") {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
	}

	// Save
	if err := gen.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving skill: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Skill saved to %s\n", gen.OutputPath)
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	text, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

func promptRequired(reader *bufio.Reader, label string) string {
	for {
		result := prompt(reader, label)
		if result != "" {
			return result
		}
		fmt.Println("This field is required.")
	}
}

func parseToolSelection(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	parts := splitAndTrim(input)
	var result []string

	for _, part := range parts {
		// Check if it's a number
		if num, err := strconv.Atoi(part); err == nil {
			if num >= 1 && num <= len(validToolNames) {
				result = append(result, validToolNames[num-1])
				continue
			}
			fmt.Fprintf(os.Stderr, "Warning: invalid tool number %d, skipping\n", num)
			continue
		}

		// Check if it's a valid tool name
		if isValidTool(part) {
			result = append(result, part)
		} else {
			fmt.Fprintf(os.Stderr, "Warning: unknown tool %q, skipping\n", part)
		}
	}

	return result
}

func isValidTool(name string) bool {
	for _, t := range validToolNames {
		if t == name {
			return true
		}
	}
	return false
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
