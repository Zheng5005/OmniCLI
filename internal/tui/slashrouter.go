package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Handler is a function that processes a slash command and returns
// an optional message to send back to the TUI, plus a command.
type Handler func(args string) (tea.Msg, tea.Cmd)

// SlashRouter dispatches slash commands to registered handlers.
type SlashRouter struct {
	handlers map[string]Handler
}

// NewSlashRouter creates a router with built-in commands.
func NewSlashRouter() *SlashRouter {
	r := &SlashRouter{
		handlers: make(map[string]Handler),
	}

	r.Register("exit", func(args string) (tea.Msg, tea.Cmd) {
		return ExitMsg{}, nil
	})

	r.Register("help", func(args string) (tea.Msg, tea.Cmd) {
		return SystemMsg{Content: "Available commands: /exit, /help, /skills, /skill <name>"}, nil
	})

	r.Register("skills", func(args string) (tea.Msg, tea.Cmd) {
		return SystemMsg{Content: "Available skills: (no skills loaded yet)"}, nil
	})

	r.Register("skill", func(args string) (tea.Msg, tea.Cmd) {
		if strings.TrimSpace(args) == "" {
			return SystemMsg{Content: "Usage: /skill <name>. Use /skills to list available skills."}, nil
		}
		return SkillActivateMsg{Name: args}, nil
	})

	return r
}

// Register adds a handler for a command (without the leading /).
func (r *SlashRouter) Register(cmd string, handler Handler) {
	r.handlers[cmd] = handler
}

// Dispatch checks if input starts with / and routes to the handler.
// Returns (handled bool, msg tea.Msg, cmd tea.Cmd).
// If not a slash command, returns (false, nil, nil).
func (r *SlashRouter) Dispatch(input string) (bool, tea.Msg, tea.Cmd) {
	if !strings.HasPrefix(input, "/") {
		return false, nil, nil
	}

	input = strings.TrimPrefix(input, "/")
	if input == "" {
		return true, SystemMsg{Content: "No command specified. Type /help for available commands."}, nil
	}

	parts := strings.SplitN(input, " ", 2)
	cmd := strings.TrimSpace(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	handler, ok := r.handlers[cmd]
	if !ok {
		return true, SystemMsg{Content: fmt.Sprintf("Unknown command: /%s", cmd)}, nil
	}

	msg, cmdFunc := handler(args)
	return true, msg, cmdFunc
}

// List returns all registered command names.
func (r *SlashRouter) List() []string {
	cmds := make([]string, 0, len(r.handlers))
	for cmd := range r.handlers {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)
	return cmds
}
