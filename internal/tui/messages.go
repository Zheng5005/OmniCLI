// Package tui implements the Bubble Tea terminal user interface for OmniCLI.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/omnicli/omnicli/internal/mcp"
)

// SubmitMsg is sent when the user submits a prompt from the input.
type SubmitMsg struct {
	Content string
}

// ApprovalRequestMsg requests user approval for a command.
// This is sent from the run_command tool's approvalFn via a channel.
type ApprovalRequestMsg struct {
	Command        string
	Classification string // "safe" or "risky"
	ResponseCh     chan bool
}

// MCPApprovalRequestMsg requests user approval for an MCP tool call from an
// untrusted server. This is sent from the MCPTool's approvalFn via a channel.
type MCPApprovalRequestMsg struct {
	ServerName  string
	ToolName    string
	Description string
	Args        string
	ResponseCh  chan bool
}

// McpServerListMsg triggers the TUI to display a list of MCP servers.
type McpServerListMsg struct{}

// McpErrorMsg displays an MCP server error in the TUI.
type McpErrorMsg struct {
	ServerName string
	Error      string
}

// McpTrustedToolFlashMsg triggers a brief status bar notification for
// a trusted MCP tool execution.
type McpTrustedToolFlashMsg struct {
	ServerName string
	ToolName   string
}

// ResourceBrowserOpenMsg triggers opening the MCP resource browser.
type ResourceBrowserOpenMsg struct{}

// ResourceListLoadedMsg carries available MCP resources for the browser.
type ResourceListLoadedMsg struct {
	Items []ResourceItem
}

// ResourceItem represents a single available MCP resource.
type ResourceItem struct {
	ServerName  string
	URI         string
	Name        string
	Description string
}

// ResourceSelectedMsg is sent when the user selects a resource in the browser.
type ResourceSelectedMsg struct {
	ServerName  string
	URI         string
	Name        string
	Description string
}

// ResourceAttachMsg carries a successfully read resource to be pinned.
type ResourceAttachMsg struct {
	Resource mcp.PinnedResource
}

// ResourceDetachMsg requests removal of a pinned resource.
type ResourceDetachMsg struct {
	ServerName string
	URI        string
}

// ResourceErrorMsg reports a resource read failure.
type ResourceErrorMsg struct {
	Err string
}

// WindowSizeMsg wraps tea.WindowSizeMsg for internal routing.
type WindowSizeMsg = tea.WindowSizeMsg

// ExitMsg signals the user wants to exit.
type ExitMsg struct{}

// SystemMsg displays a system message in the viewport.
type SystemMsg struct{ Content string }

// SkillActivateMsg requests activation of a skill by name.
type SkillActivateMsg struct{ Name string }

// SkillActivatedMsg confirms a skill was activated with resolved variables.
type SkillActivatedMsg struct{ Name, DisplayName, Prompt string }

// WizardCompleteMsg is sent when the variable wizard finishes.
type WizardCompleteMsg struct{ Values map[string]string }

// WizardCancelledMsg is sent when the user cancels the wizard.
type WizardCancelledMsg struct{}
