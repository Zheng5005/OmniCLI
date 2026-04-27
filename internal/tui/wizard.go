package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// VariableWizard is a Bubble Tea sub-model that prompts the user
// for each variable needed by a skill's system_prompt.
type VariableWizard struct {
	variables []string       // variable names in order
	descs     map[string]string // variable name -> description
	inputs    []textinput.Model
	current   int            // index of current variable being edited
	done      bool           // true when all variables are filled
	cancelled bool           // true if user cancelled
	title     string         // skill display name
	width     int
	height    int
}

// NewVariableWizard creates a wizard for the given skill variables.
func NewVariableWizard(skillName string, variables []string, descs map[string]string) VariableWizard {
	inputs := make([]textinput.Model, len(variables))
	for i, v := range variables {
		ti := textinput.New()
		ti.Placeholder = descs[v]
		if i == 0 {
			_ = ti.Focus()
		}
		inputs[i] = ti
	}

	return VariableWizard{
		variables: variables,
		descs:     descs,
		inputs:    inputs,
		current:   0,
		title:     skillName,
	}
}

// Init returns the initial command.
func (m VariableWizard) Init() tea.Cmd {
	if len(m.inputs) == 0 {
		return func() tea.Msg {
			return WizardCompleteMsg{Values: make(map[string]string)}
		}
	}
	return textinput.Blink
}

// Update handles wizard input.
func (m VariableWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.inputs[m.current].Blur()
			m.current++
			if m.current >= len(m.inputs) {
				m.done = true
				return m, func() tea.Msg {
					return WizardCompleteMsg{Values: m.collectValues()}
				}
			}
			_ = m.inputs[m.current].Focus()
			return m, textinput.Blink

		case tea.KeyEscape:
			m.cancelled = true
			return m, func() tea.Msg {
				return WizardCancelledMsg{}
			}
		}

		var cmd tea.Cmd
		m.inputs[m.current], cmd = m.inputs[m.current].Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the wizard form.
func (m VariableWizard) View() string {
	if m.done || m.cancelled || len(m.inputs) == 0 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("7")).
		MarginBottom(1)

	varStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		MarginBottom(1)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		MarginTop(1)

	currentVar := m.variables[m.current]

	form := lipgloss.JoinVertical(lipgloss.Center,
		titleStyle.Render(m.title),
		varStyle.Render(fmt.Sprintf("%s", currentVar)),
		m.inputs[m.current].View(),
		hintStyle.Render("[Enter] continue  [Esc] cancel"),
	)

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		form,
	)
}

// Values returns the collected variable values (only valid when done).
func (m VariableWizard) Values() map[string]string {
	return m.collectValues()
}

// SetSize sets the wizard dimensions.
func (m *VariableWizard) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m VariableWizard) collectValues() map[string]string {
	values := make(map[string]string, len(m.variables))
	for i, v := range m.variables {
		values[v] = m.inputs[i].Value()
	}
	return values
}
