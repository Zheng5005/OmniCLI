package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputModel wraps a Bubble Tea textarea for user prompt input.
type InputModel struct {
	textarea    textarea.Model
	enabled     bool
	skillPrefix string // e.g., "[Docs Expert] " — empty when no skill
}

// NewInputModel creates a new InputModel with sensible defaults.
func NewInputModel() InputModel {
	ta := textarea.New()
	ta.Placeholder = "Ask anything... (Enter to submit, Shift+Enter for newline)"
	ta.CharLimit = 0
	ta.MaxHeight = 5
	ta.ShowLineNumbers = false
	ta.Focus()

	return InputModel{
		textarea: ta,
		enabled:  true,
	}
}

// Update handles input messages. Enter (without modifiers) submits; all
// other keys are delegated to the underlying textarea.
func (m InputModel) Update(msg tea.Msg) (InputModel, tea.Cmd) {
	if !m.enabled {
		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEnter && !keyMsg.Alt {
			value := strings.TrimSpace(m.textarea.Value())
			if value != "" {
				m.textarea.Reset()
				return m, func() tea.Msg {
					return SubmitMsg{Content: value}
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// View renders the textarea, prepending the skill prefix when active.
func (m InputModel) View() string {
	if m.skillPrefix == "" {
		return m.textarea.View()
	}
	prefixStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	prefix := prefixStyle.Render(m.skillPrefix)
	return prefix + m.textarea.View()
}

// SetSkillPrefix updates the displayed skill prefix.
func (m *InputModel) SetSkillPrefix(prefix string) {
	m.skillPrefix = prefix
}

// SetEnabled toggles input acceptance, focusing or blurring accordingly.
func (m *InputModel) SetEnabled(enabled bool) {
	m.enabled = enabled
	if enabled {
		m.textarea.Focus()
	} else {
		m.textarea.Blur()
	}
}

// SetWidth updates the textarea width.
func (m *InputModel) SetWidth(width int) {
	m.textarea.SetWidth(width)
}

// Reset clears the textarea content and refocuses it.
func (m *InputModel) Reset() {
	m.textarea.Reset()
	m.textarea.Focus()
}
