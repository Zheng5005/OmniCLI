package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/omnicli/omnicli/internal/agent"
	"github.com/omnicli/omnicli/internal/mcp"
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
	// StateMcpApproval waits for user confirmation of an MCP tool call.
	StateMcpApproval
	// StateResourceBrowser is active while the resource picker is displayed.
	StateResourceBrowser
	// StateCommandPalette is active while the command palette is displayed.
	StateCommandPalette
)

// Model is the root Bubble Tea model composing input, viewport, and status bar.
type Model struct {
	input              InputModel
	viewport           ViewportModel
	statusBar          StatusBarModel
	agent              *agent.Agent
	state              State
	width              int
	height             int
	pendingApproval    *ApprovalRequestMsg
	pendingMCPApproval *MCPApprovalRequestMsg
	approvalDialog     ApprovalDialog
	showApprovalDialog bool
	resourcePanel      ResourcePanel
	resourceBrowser    ResourceBrowser
	showResourceBrowser bool
	ctx                context.Context
	cancel             context.CancelFunc
	router             *SlashRouter
	wizard             VariableWizard
	showWizard         bool
	commandPalette     CommandPalette
	showCommandPalette bool
	spinner            spinner.Model
	activeSkill        *skills.Skill
	skillManager       *skills.Manager
}

// NewModel creates a new root Model with the given agent, router, and skill manager.
func NewModel(agentInstance *agent.Agent, router *SlashRouter, skillManager *skills.Manager) Model {
	w, h := 80, 24
	ctx, cancel := context.WithCancel(context.Background())

	modelName := ""
	if agentInstance != nil {
		modelName = agentInstance.ModelName()
	}

	s := spinner.New()
	s.Spinner = spinner.Dot

	var palette CommandPalette
	if router != nil {
		palette = *NewCommandPalette(router.Descriptions())
	}

	return Model{
		input:           NewInputModel(),
		viewport:        NewViewportModel(w, h-inputHeight-statusBarHeight-padding),
		statusBar:       NewStatusBarModel(modelName, w),
		agent:           agentInstance,
		state:           StateNormal,
		width:           w,
		height:          h,
		resourcePanel:   NewResourcePanel(),
		resourceBrowser: NewResourceBrowser(w, h),
		commandPalette:  palette,
		spinner:         s,
		ctx:             ctx,
		cancel:          cancel,
		router:          router,
		skillManager:    skillManager,
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

// setSpinnerStyle updates the spinner color to match the given activity.
func (m *Model) setSpinnerStyle(a Activity) {
	m.spinner.Style = lipgloss.NewStyle().Foreground(a.color())
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
		case "ctrl+r":
			if m.state == StateNormal {
				m.resourcePanel.Toggle()
				m.recalcLayout()
				return m, nil
			}
		case "ctrl+p":
			if m.state == StateNormal {
				m.state = StateCommandPalette
				m.showCommandPalette = true
				m.commandPalette = *NewCommandPalette(m.router.Descriptions())
				m.commandPalette.SetSize(m.width, m.height)
				return m, m.commandPalette.Init()
			}
		}

		if m.state == StateMcpApproval {
			model, cmd := m.approvalDialog.Update(msg)
			m.approvalDialog = model.(ApprovalDialog)
			if m.approvalDialog.Approved() != nil {
				m.state = StateStreaming
				m.showApprovalDialog = false
				m.pendingMCPApproval = nil
			}
			return m, cmd
		}

		if m.state == StateResourceBrowser {
			switch msg.String() {
			case "j", "down":
				m.resourceBrowser.MoveDown()
				return m, nil
			case "k", "up":
				m.resourceBrowser.MoveUp()
				return m, nil
			case "enter":
				m.resourceBrowser.MarkSelected()
				item := m.resourceBrowser.Selected()
				if item != nil {
					m.showResourceBrowser = false
					m.state = StateNormal
					return m, m.readResourceCmd(item.ServerName, item.URI, item.Name, item.Description)
				}
				m.showResourceBrowser = false
				m.state = StateNormal
				return m, nil
			case "esc", "q":
				m.showResourceBrowser = false
				m.state = StateNormal
				return m, nil
			}
			return m, nil
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

		if m.state == StateCommandPalette {
			model, cmd := m.commandPalette.Update(msg)
			m.commandPalette = model.(CommandPalette)
			return m, cmd
		}

		if m.state == StateNormal {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}

		return m, nil

	case spinner.TickMsg:
		if m.state == StateNormal {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()
		m.statusBar.SetWidth(msg.Width)
		if m.showWizard {
			m.wizard.SetSize(msg.Width, msg.Height)
		}
		if m.showResourceBrowser {
			m.resourceBrowser.SetSize(msg.Width, msg.Height)
		}
		if m.showCommandPalette {
			m.commandPalette.SetSize(msg.Width, msg.Height)
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
		m.setSpinnerStyle(ActivityThinking)

		agentRef := m.agent
		ctx := m.ctx
		prompt := msg.Content
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg {
				go agentRef.Run(ctx, prompt)
				return nil
			},
		)

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
			m.statusBar.SetActivity(ActivityThinking)
			m.setSpinnerStyle(ActivityThinking)
			return m, tea.Batch(m.spinner.Tick, m.wizard.Init())
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
			m.setSpinnerStyle(ActivityExecuting)
		} else {
			m.statusBar.SetActivity(ActivitySearching)
			m.setSpinnerStyle(ActivitySearching)
		}
		m.viewport.AppendSystem(fmt.Sprintf("🔧 Using tool: %s", msg.Name))

		// Detect MCP tool calls and show flash for trusted servers.
		if strings.Contains(msg.Name, "__") {
			parts := strings.SplitN(msg.Name, "__", 2)
			if len(parts) == 2 && m.agent != nil && m.agent.MCPManager() != nil {
				client := m.agent.MCPManager().Client(parts[0])
				if client != nil && client.Config().Trusted {
					m.statusBar.SetFlash(fmt.Sprintf("[✓] %s:%s", parts[0], parts[1]), 2*time.Second)
				}
			}
		}
		m.updateMCPStatus()
		return m, nil

	case agent.AgentDoneMsg:
		m.viewport.FinalizeResponse()
		m.state = StateNormal
		m.input.SetEnabled(true)
		m.statusBar.SetActivity(ActivityReady)
		m.updateMCPStatus()
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
		m.setSpinnerStyle(ActivityExecuting)
		return m, nil

	case MCPApprovalRequestMsg:
		m.state = StateMcpApproval
		m.pendingMCPApproval = &msg
		m.approvalDialog = NewApprovalDialog(msg.ServerName, msg.ToolName, msg.Description, msg.Args, msg.ResponseCh)
		m.approvalDialog.SetSize(m.width, m.height)
		m.showApprovalDialog = true
		m.statusBar.SetActivity(ActivityExecuting)
		m.setSpinnerStyle(ActivityExecuting)
		return m, nil

	case McpServerListMsg:
		return m.handleMcpServerList()

	case McpErrorMsg:
		m.viewport.AppendSystem(fmt.Sprintf("⚠️ MCP Server '%s' error: %s — check /mcp for details", msg.ServerName, msg.Error))
		m.updateMCPStatus()
		return m, nil

	case McpTrustedToolFlashMsg:
		m.statusBar.SetFlash(fmt.Sprintf("[✓] %s:%s", msg.ServerName, msg.ToolName), 2*time.Second)
		return m, nil

	case ResourceBrowserOpenMsg:
		m.state = StateResourceBrowser
		m.showResourceBrowser = true
		m.resourceBrowser = NewResourceBrowser(m.width, m.height)
		m.statusBar.SetActivity(ActivitySearching)
		m.setSpinnerStyle(ActivitySearching)
		return m, tea.Batch(m.spinner.Tick, m.fetchResourcesCmd())

	case ResourceListLoadedMsg:
		m.resourceBrowser.SetItems(msg.Items)
		return m, nil

	case ResourceSelectedMsg:
		return m, m.readResourceCmd(msg.ServerName, msg.URI, msg.Name, msg.Description)

	case ResourceAttachMsg:
		if m.agent != nil {
			_ = m.agent.AttachResource(msg.Resource)
			if err := m.agent.RecreateSession("", m.agent.AllToolNames(), nil); err != nil {
				m.viewport.AppendSystem(fmt.Sprintf("⚠️ Failed to update session: %v", err))
			}
		}
		m.updateResourcePanel()
		m.viewport.AppendSystem(fmt.Sprintf("📎 Attached resource: %s (%s)", msg.Resource.Name, msg.Resource.URI))
		return m, nil

	case ResourceDetachMsg:
		m.detachResource(msg.ServerName, msg.URI)
		return m, nil

	case ResourceErrorMsg:
		m.viewport.AppendSystem(fmt.Sprintf("⚠️ Resource error: %s", msg.Err))
		return m, nil

	case CommandPaletteDismissMsg:
		m.showCommandPalette = false
		m.state = StateNormal
		return m, nil

	case CommandPaletteExecuteMsg:
		m.showCommandPalette = false
		m.state = StateNormal
		handled, routerMsg, cmd := m.router.Dispatch("/" + msg.Command)
		if handled {
			if routerMsg != nil {
				return m, func() tea.Msg {
					return routerMsg
				}
			}
			return m, cmd
		}
		return m, nil
	}

	// Delegate scroll events to viewport.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleApprovalKey processes key presses during the approval state.
func (m Model) handleApprovalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle MCP tool approval.
	if m.pendingMCPApproval != nil {
		switch {
		case msg.String() == "y":
			m.pendingMCPApproval.ResponseCh <- true
		case msg.String() == "n":
			m.pendingMCPApproval.ResponseCh <- false
		default:
			return m, nil
		}
		m.pendingMCPApproval = nil
		m.state = StateStreaming
		return m, nil
	}

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

// handleMcpServerList builds and displays the MCP server list in the viewport.
func (m Model) handleMcpServerList() (tea.Model, tea.Cmd) {
	if m.agent == nil || m.agent.MCPManager() == nil {
		m.viewport.AppendSystem("No MCP manager configured.")
		return m, nil
	}

	mgr := m.agent.MCPManager()
	statuses := mgr.Status()
	if len(statuses) == 0 {
		m.viewport.AppendSystem("No MCP servers configured.")
		return m, nil
	}

	var b strings.Builder
	b.WriteString("## MCP Servers\n\n")
	b.WriteString("| Server | Status | Tools | Health |\n")
	b.WriteString("|--------|--------|-------|--------|\n")
	for name, st := range statuses {
		health := "✓"
		if !st.Healthy {
			health = "✗"
		}
		stateIcon := "🟢"
		switch st.State {
		case "error", "failed":
			stateIcon = "🔴"
		case "disconnected":
			stateIcon = "⚪"
		}
		b.WriteString(fmt.Sprintf("| %s | %s %s | %d | %s |\n", name, stateIcon, st.State, st.Tools, health))
		if st.Error != "" {
			b.WriteString(fmt.Sprintf("| | *Error: %s* | | |\n", st.Error))
		}
	}
	b.WriteString("\n")

	for name, st := range statuses {
		if st.State != "ready" {
			continue
		}
		client := mgr.Client(name)
		if client == nil {
			continue
		}
		tools := client.Tools()
		if len(tools) > 0 {
			b.WriteString(fmt.Sprintf("### %s Tools\n\n", name))
			for _, t := range tools {
				b.WriteString(fmt.Sprintf("- **%s**: %s\n", t.Name, t.Description))
			}
			b.WriteString("\n")
		}
	}

	m.viewport.AppendMarkdown(b.String())
	m.updateMCPStatus()
	return m, nil
}

// recalcLayout adjusts viewport and panel sizes based on current dimensions.
func (m *Model) recalcLayout() {
	panelWidth := 0
	if m.resourcePanel.Visible() {
		panelWidth = m.width / 3
		if panelWidth > 40 {
			panelWidth = 40
		}
		if panelWidth < 20 {
			panelWidth = 20
		}
	}
	vpWidth := m.width - panelWidth
	vpHeight := m.height - inputHeight - statusBarHeight - padding
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport.SetSize(vpWidth, vpHeight)
	m.resourcePanel.SetSize(panelWidth, vpHeight)
	m.input.SetWidth(vpWidth)
}

// fetchResourcesCmd returns a command that fetches available resources from all MCP servers.
func (m Model) fetchResourcesCmd() tea.Cmd {
	return func() tea.Msg {
		if m.agent == nil || m.agent.MCPManager() == nil {
			return ResourceListLoadedMsg{Items: nil}
		}
		mgr := m.agent.MCPManager()
		var items []ResourceItem
		for name, st := range mgr.Status() {
			if st.State != "ready" {
				continue
			}
			client := mgr.Client(name)
			if client == nil {
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			resources, err := client.ListResources(ctx)
			cancel()
			if err != nil {
				continue
			}
			for _, r := range resources {
				items = append(items, ResourceItem{
					ServerName:  name,
					URI:         r.URI,
					Name:        r.Name,
					Description: r.Description,
				})
			}
		}
		return ResourceListLoadedMsg{Items: items}
	}
}

// readResourceCmd returns a command that reads an MCP resource and returns an attach message.
func (m Model) readResourceCmd(serverName, uri, name, description string) tea.Cmd {
	return func() tea.Msg {
		if m.agent == nil || m.agent.MCPManager() == nil {
			return ResourceErrorMsg{Err: "MCP manager not available"}
		}
		client := m.agent.MCPManager().Client(serverName)
		if client == nil {
			return ResourceErrorMsg{Err: fmt.Sprintf("server %s not connected", serverName)}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		content, err := client.ReadResource(ctx, uri)
		cancel()
		if err != nil {
			return ResourceErrorMsg{Err: fmt.Sprintf("failed to read %s: %v", uri, err)}
		}
		return ResourceAttachMsg{
			Resource: mcp.PinnedResource{
				ServerName:  serverName,
				URI:         uri,
				Name:        name,
				Description: description,
				Content:     content,
			},
		}
	}
}

// detachResource removes a pinned resource and updates the session.
func (m *Model) detachResource(serverName, uri string) {
	if m.agent != nil {
		// If serverName is empty, try to find by URI only.
		if serverName == "" {
			for _, r := range m.agent.PinnedResources() {
				if r.URI == uri {
					serverName = r.ServerName
					break
				}
			}
		}
		m.agent.DetachResource(serverName, uri)
		if err := m.agent.RecreateSession("", m.agent.AllToolNames(), nil); err != nil {
			m.viewport.AppendSystem(fmt.Sprintf("⚠️ Failed to update session: %v", err))
		}
	}
	m.updateResourcePanel()
	m.viewport.AppendSystem(fmt.Sprintf("📎 Detached resource: %s", uri))
}

// updateResourcePanel syncs the panel with the agent's pinned resources.
func (m *Model) updateResourcePanel() {
	if m.agent != nil {
		m.resourcePanel.SetResources(m.agent.PinnedResources())
	}
}

// updateMCPStatus queries the MCP manager and updates the status bar.
func (m *Model) updateMCPStatus() {
	if m.agent == nil || m.agent.MCPManager() == nil {
		m.statusBar.SetMCPStatus("")
		return
	}

	statuses := m.agent.MCPManager().Status()
	if len(statuses) == 0 {
		m.statusBar.SetMCPStatus("")
		return
	}

	active := 0
	errors := 0
	for _, st := range statuses {
		if st.State == "ready" {
			active++
		} else if st.State == "error" || st.State == "failed" {
			errors++
		}
	}

	var parts []string
	if active > 0 {
		parts = append(parts, fmt.Sprintf("MCP:%d↑", active))
	}
	if errors > 0 {
		parts = append(parts, fmt.Sprintf("%d⚠", errors))
	}

	if len(parts) > 0 {
		m.statusBar.SetMCPStatus("[" + strings.Join(parts, " ") + "]")
	} else {
		m.statusBar.SetMCPStatus("")
	}
}

// View renders the full TUI layout.
func (m Model) View() string {
	spinnerView := ""
	if m.state != StateNormal {
		spinnerView = m.spinner.View()
	}

	var view string
	if m.resourcePanel.Visible() {
		view = lipgloss.JoinHorizontal(lipgloss.Top,
			m.viewport.View(),
			m.resourcePanel.View(),
		)
		view = lipgloss.JoinVertical(lipgloss.Left,
			view,
			m.statusBar.View(spinnerView),
			m.input.View(),
		)
	} else {
		view = lipgloss.JoinVertical(lipgloss.Left,
			m.viewport.View(),
			m.statusBar.View(spinnerView),
			m.input.View(),
		)
	}

	if m.showWizard {
		wizardOverlay := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.wizard.View(),
		)
		return wizardOverlay
	}

	if m.showApprovalDialog {
		approvalOverlay := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.approvalDialog.View(),
		)
		return approvalOverlay
	}

	if m.showResourceBrowser {
		browserOverlay := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.resourceBrowser.View(),
		)
		return browserOverlay
	}

	if m.showCommandPalette {
		paletteOverlay := lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.commandPalette.View(),
		)
		return paletteOverlay
	}

	return view
}
