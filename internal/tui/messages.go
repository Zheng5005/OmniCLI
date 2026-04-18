// Package tui implements the Bubble Tea terminal user interface for OmniCLI.
package tui

import tea "github.com/charmbracelet/bubbletea"

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

// WindowSizeMsg wraps tea.WindowSizeMsg for internal routing.
type WindowSizeMsg = tea.WindowSizeMsg
