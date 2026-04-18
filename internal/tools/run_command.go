package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/omnicli/omnicli/internal/exec"
)

// RunCommandTool implements the Tool interface for executing shell commands.
type RunCommandTool struct {
	safePatterns []*regexp.Regexp
	approvalFn   func(cmd string, classification exec.Classification) (bool, error)
}

// NewRunCommandTool creates a new RunCommandTool with the given safe patterns
// and an approval callback used to request user confirmation before execution.
func NewRunCommandTool(patterns []*regexp.Regexp, approvalFn func(string, exec.Classification) (bool, error)) *RunCommandTool {
	return &RunCommandTool{
		safePatterns: patterns,
		approvalFn:   approvalFn,
	}
}

// Name returns the tool name.
func (t *RunCommandTool) Name() string { return "run_command" }

// Description returns a human-readable description of the tool.
func (t *RunCommandTool) Description() string {
	return "Execute a shell command and return its output."
}

// Parameters returns the JSON Schema for the tool's arguments.
func (t *RunCommandTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object","required":["command"],"properties":{"command":{"type":"string","description":"Shell command to execute"},"timeout":{"type":"integer","description":"Timeout in seconds (default 30)"}}}`)
}

type runCommandArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

// Execute runs the shell command after classification and approval.
func (t *RunCommandTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a runCommandArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	timeout := 30
	if a.Timeout > 0 {
		timeout = a.Timeout
	}

	classification := exec.Classify(a.Command, t.safePatterns)

	approved, err := t.approvalFn(a.Command, classification)
	if err != nil {
		return "", fmt.Errorf("approval error: %w", err)
	}
	if !approved {
		return fmt.Sprintf("Command denied by user: %s", a.Command), nil
	}

	result, err := exec.Run(ctx, a.Command, time.Duration(timeout)*time.Second)
	if err != nil {
		return "", fmt.Errorf("execution error: %w", err)
	}

	var b strings.Builder
	b.WriteString(result.Stdout)
	if result.Stderr != "" {
		b.WriteString("\nSTDERR:\n")
		b.WriteString(result.Stderr)
	}
	if result.ExitCode != 0 {
		fmt.Fprintf(&b, "\nExit code: %d", result.ExitCode)
	}
	return b.String(), nil
}
