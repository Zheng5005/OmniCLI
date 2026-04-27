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
