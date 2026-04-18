package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnicli/omnicli/internal/agent"
)

const (
	inputHeight    = 3
	statusBarHeight = 1
	padding        = 2
)

// State represents the TUI state machine.
type State int

const (
	// StateNormal is the idle state awaiting user input.
	StateNormal State = iota
	// StateStreaming is active while the agent is producing output.
	StateStreaming
	// StateAwaitingApproval waits for user confirmation of a command.
	StateAwaitingApproval
)

// Model is the root Bubble Tea model composing input, viewport, and status bar.
type Model struct {
	input           InputModel
	viewport        ViewportModel
	statusBar       StatusBarModel
	agent           *agent.Agent
	state           State
	width           int
	height          int
	pendingApproval *ApprovalRequestMsg
	ctx             context.Context
	cancel          context.CancelFunc
}

// NewModel creates a new root Model with the given agent.
func NewModel(agentInstance *agent.Agent) Model {
	w, h := 80, 24
	ctx, cancel := context.WithCancel(context.Background())

	modelName := ""
	if agentInstance != nil {
		modelName = agentInstance.ModelName()
	}

	return Model{
		input:     NewInputModel(),
		viewport:  NewViewportModel(w, h-inputHeight-statusBarHeight-padding),
		statusBar: NewStatusBarModel(modelName, w),
		agent:     agentInstance,
		state:     StateNormal,
		width:     w,
		height:    h,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Init returns the initial command to query terminal dimensions.
func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

// Update processes incoming messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.cancel()
			return m, tea.Quit
		}

		if msg.Type == tea.KeyRunes && string(msg.Runes) == "/exit" {
			m.cancel()
			return m, tea.Quit
		}

		if m.state == StateAwaitingApproval {
			return m.handleApprovalKey(msg)
		}

		if m.state == StateNormal {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpHeight := msg.Height - inputHeight - statusBarHeight - padding
		if vpHeight < 1 {
			vpHeight = 1
		}
		m.viewport.SetSize(msg.Width, vpHeight)
		m.statusBar.SetWidth(msg.Width)
		return m, nil

	case SubmitMsg:
		m.viewport.AppendSystem("> " + msg.Content)
		m.state = StateStreaming
		m.input.SetEnabled(false)
		m.statusBar.SetActivity(ActivityThinking)

		agentRef := m.agent
		ctx := m.ctx
		prompt := msg.Content
		return m, func() tea.Msg {
			go agentRef.Run(ctx, prompt)
			return nil
		}

	case agent.StreamChunkMsg:
		m.viewport.AppendChunk(msg.Content)
		return m, nil

	case agent.ToolCallMsg:
		if msg.Name == "run_command" {
			m.statusBar.SetActivity(ActivityExecuting)
		} else {
			m.statusBar.SetActivity(ActivitySearching)
		}
		m.viewport.AppendSystem(fmt.Sprintf("🔧 Using tool: %s", msg.Name))
		return m, nil

	case agent.AgentDoneMsg:
		m.viewport.FinalizeResponse()
		m.state = StateNormal
		m.input.SetEnabled(true)
		m.statusBar.SetActivity(ActivityReady)
		return m, nil

	case agent.CostUpdateMsg:
		m.statusBar.SetCost(msg.Cost)
		return m, nil

	case agent.ErrorMsg:
		m.viewport.AppendSystem(fmt.Sprintf("⚠️ Error: %v", msg.Err))
		m.state = StateNormal
		m.input.SetEnabled(true)
		m.statusBar.SetActivity(ActivityReady)
		return m, nil

	case ApprovalRequestMsg:
		m.state = StateAwaitingApproval
		m.pendingApproval = &msg
		m.viewport.AppendSystem(
			fmt.Sprintf("⚡ Command: %s [%s] — Press [Enter/y] to run, [n] to deny",
				msg.Command, msg.Classification),
		)
		m.statusBar.SetActivity(ActivityExecuting)
		return m, nil
	}

	// Delegate scroll events to viewport.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleApprovalKey processes key presses during the approval state.
func (m Model) handleApprovalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.pendingApproval == nil {
		return m, nil
	}

	switch {
	case msg.Type == tea.KeyEnter:
		if m.pendingApproval.Classification == "safe" {
			m.pendingApproval.ResponseCh <- true
		} else {
			return m, nil
		}
	case msg.String() == "y":
		m.pendingApproval.ResponseCh <- true
	case msg.String() == "n":
		m.pendingApproval.ResponseCh <- false
	default:
		return m, nil
	}

	m.pendingApproval = nil
	m.state = StateStreaming
	return m, nil
}

// View renders the full TUI layout.
func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.viewport.View(),
		m.statusBar.View(),
		m.input.View(),
	)
}
