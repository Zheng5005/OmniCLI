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

// CommandDesc pairs a command name with its description.
type CommandDesc struct {
	Name        string
	Description string
}

// SlashRouter dispatches slash commands to registered handlers.
type SlashRouter struct {
	handlers     map[string]Handler
	descriptions map[string]string
}

// NewSlashRouter creates a router with built-in commands.
func NewSlashRouter() *SlashRouter {
	r := &SlashRouter{
		handlers:     make(map[string]Handler),
		descriptions: make(map[string]string),
	}

	r.Register("exit", func(args string) (tea.Msg, tea.Cmd) {
		return ExitMsg{}, nil
	})
	r.descriptions["exit"] = "Exit the application"

	r.Register("help", func(args string) (tea.Msg, tea.Cmd) {
		var b strings.Builder
		b.WriteString("Available commands:\n")
		for _, desc := range r.Descriptions() {
			b.WriteString(fmt.Sprintf("  /%s — %s\n", desc.Name, desc.Description))
		}
		b.WriteString("\nShortcuts:\n  Ctrl+P — Open command palette")
		return SystemMsg{Content: b.String()}, nil
	})
	r.descriptions["help"] = "Show this help message"

	r.Register("mcp", func(args string) (tea.Msg, tea.Cmd) {
		return McpServerListMsg{}, nil
	})
	r.descriptions["mcp"] = "List MCP servers and their tools"

	r.Register("attach", func(args string) (tea.Msg, tea.Cmd) {
		return ResourceBrowserOpenMsg{}, nil
	})
	r.descriptions["attach"] = "Open the resource browser to attach an MCP resource"

	r.Register("detach", func(args string) (tea.Msg, tea.Cmd) {
		arg := strings.TrimSpace(args)
		if arg == "" {
			return SystemMsg{Content: "Usage: /detach <server> <uri> or /detach <uri>"}, nil
		}
		parts := strings.Fields(arg)
		if len(parts) >= 2 {
			return ResourceDetachMsg{ServerName: parts[0], URI: parts[1]}, nil
		}
		return ResourceDetachMsg{URI: parts[0]}, nil
	})
	r.descriptions["detach"] = "Detach a pinned MCP resource"

	r.Register("skills", func(args string) (tea.Msg, tea.Cmd) {
		return SystemMsg{Content: "Available skills: (no skills loaded yet)"}, nil
	})
	r.descriptions["skills"] = "List available skills"

	r.Register("skill", func(args string) (tea.Msg, tea.Cmd) {
		if strings.TrimSpace(args) == "" {
			return SystemMsg{Content: "Usage: /skill <name>. Use /skills to list available skills."}, nil
		}
		return SkillActivateMsg{Name: args}, nil
	})
	r.descriptions["skill"] = "Activate a skill by name"

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

// Descriptions returns all registered commands with their descriptions.
func (r *SlashRouter) Descriptions() []CommandDesc {
	cmds := r.List()
	descs := make([]CommandDesc, 0, len(cmds))
	for _, cmd := range cmds {
		desc := r.descriptions[cmd]
		if desc == "" {
			desc = "No description available."
		}
		descs = append(descs, CommandDesc{Name: cmd, Description: desc})
	}
	return descs
}
