package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

// InputModel wraps a Bubble Tea textarea for user prompt input.
type InputModel struct {
	textarea textarea.Model
	enabled  bool
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
			value := m.textarea.Value()
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

// View renders the textarea.
func (m InputModel) View() string {
	return m.textarea.View()
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

// Reset clears the textarea content and refocuses it.
func (m *InputModel) Reset() {
	m.textarea.Reset()
	m.textarea.Focus()
}
