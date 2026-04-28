package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ApprovalDialog renders an MCP tool approval overlay with server name,
// tool name, description, formatted arguments, and Approve/Deny buttons.
type ApprovalDialog struct {
	serverName  string
	toolName    string
	description string
	args        string
	responseCh  chan bool
	focused     int // 0 = Approve, 1 = Deny
	width       int
	height      int
	approved    *bool
}

// NewApprovalDialog creates a new approval dialog for the given MCP tool call.
func NewApprovalDialog(serverName, toolName, description, args string, responseCh chan bool) ApprovalDialog {
	return ApprovalDialog{
		serverName:  serverName,
		toolName:    toolName,
		description: description,
		args:        args,
		responseCh:  responseCh,
		focused:     0,
	}
}

// SetSize updates the dialog dimensions.
func (m *ApprovalDialog) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Init returns the initial command.
func (m ApprovalDialog) Init() tea.Cmd {
	return nil
}

// Update handles keyboard navigation and selection.
func (m ApprovalDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			approved := m.focused == 0
			m.approved = &approved
			if m.responseCh != nil {
				m.responseCh <- approved
			}
			return m, nil
		case tea.KeyTab, tea.KeyRight:
			m.focused = 1
			return m, nil
		case tea.KeyShiftTab, tea.KeyLeft:
			m.focused = 0
			return m, nil
		}
		switch msg.String() {
		case "y":
			approved := true
			m.approved = &approved
			if m.responseCh != nil {
				m.responseCh <- true
			}
			return m, nil
		case "n":
			approved := false
			m.approved = &approved
			if m.responseCh != nil {
				m.responseCh <- false
			}
			return m, nil
		case "j":
			m.focused = 1
			return m, nil
		case "k":
			m.focused = 0
			return m, nil
		}
	}
	return m, nil
}

// Approved returns the approval result, or nil if not decided yet.
func (m ApprovalDialog) Approved() *bool {
	return m.approved
}

// View renders the approval dialog as a centered overlay.
func (m ApprovalDialog) View() string {
	if m.approved != nil {
		return ""
	}

	boxWidth := m.width - 4
	if boxWidth > 80 {
		boxWidth = 80
	}
	if boxWidth < 40 {
		boxWidth = 40
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("3")).
		MarginBottom(1)

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("7"))

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("248"))

	argsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)

	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		MarginTop(1)

	approveStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		MarginRight(2)

	denyStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 2).
		MarginLeft(2)

	if m.focused == 0 {
		approveStyle = approveStyle.
			Background(lipgloss.Color("2")).
			Foreground(lipgloss.Color("0"))
	} else {
		approveStyle = approveStyle.
			Foreground(lipgloss.Color("2"))
	}

	if m.focused == 1 {
		denyStyle = denyStyle.
			Background(lipgloss.Color("1")).
			Foreground(lipgloss.Color("0"))
	} else {
		denyStyle = denyStyle.
			Foreground(lipgloss.Color("1"))
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("⚡ MCP Tool Approval") + "\n")
	b.WriteString(labelStyle.Render("Server: ") + valueStyle.Render(m.serverName) + "\n")
	b.WriteString(labelStyle.Render("Tool: ") + valueStyle.Render(m.toolName) + "\n")
	if m.description != "" {
		b.WriteString(labelStyle.Render("Description: ") + valueStyle.Render(m.description) + "\n")
	}
	b.WriteString(labelStyle.Render("Arguments:") + "\n")
	b.WriteString(argsStyle.Render(m.args) + "\n")

	buttons := lipgloss.JoinHorizontal(lipgloss.Center,
		approveStyle.Render("[ Approve ]"),
		denyStyle.Render("[  Deny   ]"),
	)
	b.WriteString(buttons + "\n")
	b.WriteString(hintStyle.Render("[Tab/←/→] navigate  [Enter] select  [y/n] quick action") + "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("3")).
		Padding(1, 2).
		Width(boxWidth)

	box := boxStyle.Render(b.String())

	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}
