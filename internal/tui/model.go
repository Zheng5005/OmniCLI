package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnicli/omnicli/internal/agent"
	"github.com/omnicli/omnicli/internal/skills"
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
	// StateWizard is active while the variable input wizard is displayed.
	StateWizard
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
	router          *SlashRouter
	wizard          VariableWizard
	showWizard      bool
	activeSkill     *skills.Skill
	skillManager    *skills.Manager
}

// NewModel creates a new root Model with the given agent, router, and skill manager.
func NewModel(agentInstance *agent.Agent, router *SlashRouter, skillManager *skills.Manager) Model {
	w, h := 80, 24
	ctx, cancel := context.WithCancel(context.Background())

	modelName := ""
	if agentInstance != nil {
		modelName = agentInstance.ModelName()
	}

	return Model{
		input:        NewInputModel(),
		viewport:     NewViewportModel(w, h-inputHeight-statusBarHeight-padding),
		statusBar:    NewStatusBarModel(modelName, w),
		agent:        agentInstance,
		state:        StateNormal,
		width:        w,
		height:       h,
		ctx:          ctx,
		cancel:       cancel,
		router:       router,
		skillManager: skillManager,
	}
}

// HistoryEntry represents a message from a previous session for --resume.
type HistoryEntry struct {
	Role    string
	Content string
}

// LoadHistory renders previous session messages into the viewport.
func (m *Model) LoadHistory(entries []HistoryEntry) {
	m.viewport.AppendSystem("📂 Resumed session")
	for _, e := range entries {
		switch e.Role {
		case "user":
			m.viewport.AppendSystem("> " + e.Content)
		case "assistant":
			m.viewport.AppendChunk(e.Content)
			m.viewport.FinalizeResponse()
		}
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

		if m.state == StateAwaitingApproval {
			return m.handleApprovalKey(msg)
		}

		if m.state == StateWizard {
			var cmd tea.Cmd
			var model tea.Model
			model, cmd = m.wizard.Update(msg)
			m.wizard = model.(VariableWizard)
			return m, cmd
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
		if m.showWizard {
			m.wizard.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case SubmitMsg:
		if strings.HasPrefix(msg.Content, "/") {
			handled, routerMsg, cmd := m.router.Dispatch(msg.Content)
			if handled {
				if routerMsg != nil {
					return m, func() tea.Msg {
						return routerMsg
					}
				}
				return m, cmd
			}
		}

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

	case ExitMsg:
		m.cancel()
		return m, tea.Quit

	case SystemMsg:
		m.viewport.AppendSystem(msg.Content)
		return m, nil

	case SkillActivateMsg:
		if m.skillManager == nil {
			m.viewport.AppendSystem("⚠️ Skill manager not available")
			return m, nil
		}
		if msg.Name == "none" {
			m.activateSkill(nil, nil)
			m.viewport.AppendSystem("Skill deactivated")
			return m, nil
		}
		skill, err := m.skillManager.Get(msg.Name)
		if err != nil {
			m.viewport.AppendSystem(fmt.Sprintf("⚠️ Skill not found: %s", msg.Name))
			return m, nil
		}
		m.activeSkill = skill
		vars := skill.ExtractVariables()
		if len(vars) > 0 {
			descs := skill.Variables
			if descs == nil {
				descs = make(map[string]string)
			}
			m.wizard = NewVariableWizard(skill.DisplayName, vars, descs)
			m.wizard.SetSize(m.width, m.height)
			m.showWizard = true
			m.state = StateWizard
			return m, m.wizard.Init()
		}
		m.activateSkill(skill, make(map[string]string))
		return m, nil

	case WizardCompleteMsg:
		m.showWizard = false
		m.state = StateNormal
		if m.activeSkill != nil {
			m.activateSkill(m.activeSkill, msg.Values)
		}
		return m, nil

	case WizardCancelledMsg:
		m.showWizard = false
		m.state = StateNormal
		m.activeSkill = nil
		m.viewport.AppendSystem("Skill activation cancelled")
		return m, nil

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
		hint := "[y] to run, [n] to deny"
		if msg.Classification == "safe" {
			hint = "[Enter/y] to run, [n] to deny"
		}
		m.viewport.AppendSystem(
			fmt.Sprintf("⚡ Command: %s [%s] — %s",
				msg.Command, msg.Classification, hint),
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

// activateSkill injects variables into the skill's system prompt, recreates
// the agent session with the skill's tools, and updates the UI.
func (m *Model) activateSkill(skill *skills.Skill, values map[string]string) {
	if skill == nil {
		m.activeSkill = nil
		m.statusBar.SetSkill("")
		m.input.SetSkillPrefix("")
		return
	}

	prompt, err := skill.InjectVariables(values)
	if err != nil {
		m.viewport.AppendSystem(fmt.Sprintf("⚠️ Skill activation failed: %v", err))
		m.activeSkill = nil
		m.statusBar.SetSkill("")
		m.input.SetSkillPrefix("")
		return
	}

	if m.agent == nil {
		m.viewport.AppendSystem("⚠️ Agent not available")
		m.activeSkill = nil
		m.statusBar.SetSkill("")
		m.input.SetSkillPrefix("")
		return
	}

	err = m.agent.RecreateSession(prompt, skill.Tools, nil)
	if err != nil {
		m.viewport.AppendSystem(fmt.Sprintf("⚠️ Failed to activate skill: %v", err))
		m.activeSkill = nil
		m.statusBar.SetSkill("")
		m.input.SetSkillPrefix("")
		return
	}

	m.activeSkill = skill
	m.statusBar.SetModel(m.agent.ModelName())
	m.statusBar.SetSkill(skill.DisplayName)
	m.input.SetSkillPrefix(fmt.Sprintf("[%s] ", skill.DisplayName))
	m.viewport.AppendSystem(fmt.Sprintf("✅ Skill activated: %s", skill.DisplayName))
}

// View renders the full TUI layout.
func (m Model) View() string {
	view := lipgloss.JoinVertical(lipgloss.Left,
		m.viewport.View(),
		m.statusBar.View(),
		m.input.View(),
	)

	if m.showWizard {
		wizardOverlay := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.wizard.View(),
		)
		return wizardOverlay
	}

	return view
}
