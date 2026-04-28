package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Activity represents the current agent activity state.
type Activity int

const (
	// ActivityReady indicates the agent is idle and awaiting input.
	ActivityReady Activity = iota
	// ActivityThinking indicates the agent is processing with the LLM.
	ActivityThinking
	// ActivitySearching indicates the agent is invoking a search tool.
	ActivitySearching
	// ActivityExecuting indicates the agent is executing a command.
	ActivityExecuting
)

// String returns the human-readable label for the activity.
func (a Activity) String() string {
	switch a {
	case ActivityThinking:
		return "Thinking"
	case ActivitySearching:
		return "Searching"
	case ActivityExecuting:
		return "Executing"
	default:
		return "Ready"
	}
}

func (a Activity) color() lipgloss.Color {
	switch a {
	case ActivityThinking:
		return lipgloss.Color("3") // yellow
	case ActivitySearching:
		return lipgloss.Color("6") // cyan
	case ActivityExecuting:
		return lipgloss.Color("5") // magenta
	default:
		return lipgloss.Color("2") // green
	}
}

// StatusBarModel renders a full-width status bar with model name, activity, and cost.
type StatusBarModel struct {
	model       string
	cost        float64
	activity    Activity
	width       int
	skill       string
	mcpStatus   string
	flash       string
	flashExpiry time.Time
}

// NewStatusBarModel creates a new status bar for the given model name and width.
func NewStatusBarModel(modelName string, width int) StatusBarModel {
	return StatusBarModel{
		model: modelName,
		width: width,
	}
}

// SetActivity updates the displayed activity.
func (m *StatusBarModel) SetActivity(a Activity) { m.activity = a }

// SetCost updates the displayed cumulative cost.
func (m *StatusBarModel) SetCost(cost float64) { m.cost = cost }

// SetWidth updates the bar width.
func (m *StatusBarModel) SetWidth(width int) { m.width = width }

// SetModel updates the displayed model name.
func (m *StatusBarModel) SetModel(name string) { m.model = name }

// SetSkill updates the displayed skill name.
func (m *StatusBarModel) SetSkill(name string) { m.skill = name }

// SetMCPStatus updates the displayed MCP server status summary.
func (m *StatusBarModel) SetMCPStatus(status string) { m.mcpStatus = status }

// SetFlash sets a brief flash message to display in the status bar.
func (m *StatusBarModel) SetFlash(text string, duration time.Duration) {
	m.flash = text
	m.flashExpiry = time.Now().Add(duration)
}

// View renders the status bar spanning the full terminal width.
func (m StatusBarModel) View() string {
	bg := lipgloss.Color("236")

	leftStyle := lipgloss.NewStyle().
		Bold(true).
		Background(bg).
		Foreground(lipgloss.Color("7")).
		Padding(0, 1)

	centerStyle := lipgloss.NewStyle().
		Background(bg).
		Foreground(m.activity.color()).
		Padding(0, 1)

	rightStyle := lipgloss.NewStyle().
		Background(bg).
		Foreground(lipgloss.Color("7")).
		Padding(0, 1)

	leftText := m.model
	if m.mcpStatus != "" {
		leftText += " " + m.mcpStatus
	}
	left := leftStyle.Render(leftText)

	centerText := m.activity.String()
	if m.flash != "" && time.Now().Before(m.flashExpiry) {
		centerText = m.flash
	} else {
		m.flash = ""
	}
	center := centerStyle.Render(centerText)

	rightContent := fmt.Sprintf("$%.4f", m.cost)
	if m.skill != "" {
		rightContent = fmt.Sprintf("SKILL: %s $%.4f", m.skill, m.cost)
	}
	right := rightStyle.Render(rightContent)

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	centerW := lipgloss.Width(center)

	gap := m.width - leftW - centerW - rightW
	if gap < 0 {
		gap = 0
	}

	leftGap := gap / 2
	rightGap := gap - leftGap

	filler := lipgloss.NewStyle().Background(bg)

	return left +
		filler.Render(repeatSpace(leftGap)) +
		center +
		filler.Render(repeatSpace(rightGap)) +
		right
}

func repeatSpace(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
