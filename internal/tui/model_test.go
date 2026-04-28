package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/omnicli/omnicli/internal/agent"
	"github.com/omnicli/omnicli/internal/mcp"
)

func TestModelUpdate(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Model)
		msg       tea.Msg
		wantState State
		checkFn   func(t *testing.T, m Model, cmd tea.Cmd)
	}{
		{
			name: "window resize",
			msg:  tea.WindowSizeMsg{Width: 120, Height: 40},
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.width != 120 {
					t.Errorf("width = %d, want 120", m.width)
				}
				if m.height != 40 {
					t.Errorf("height = %d, want 40", m.height)
				}
			},
		},
		{
			name:      "submit message sets streaming state",
			msg:       SubmitMsg{Content: "hello"},
			wantState: StateStreaming,
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.state != StateStreaming {
					t.Errorf("state = %d, want StateStreaming", m.state)
				}
			},
		},
		{
			name: "stream chunk stays streaming",
			setup: func(m *Model) {
				m.state = StateStreaming
			},
			msg:       agent.StreamChunkMsg{Content: "chunk"},
			wantState: StateStreaming,
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.state != StateStreaming {
					t.Errorf("state = %d, want StateStreaming", m.state)
				}
			},
		},
		{
			name: "agent done returns to normal",
			setup: func(m *Model) {
				m.state = StateStreaming
			},
			msg:       agent.AgentDoneMsg{Content: "done", TotalCost: 0.05},
			wantState: StateNormal,
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.state != StateNormal {
					t.Errorf("state = %d, want StateNormal", m.state)
				}
			},
		},
		{
			name: "error message returns to normal",
			setup: func(m *Model) {
				m.state = StateStreaming
			},
			msg:       agent.ErrorMsg{Err: fmt.Errorf("test error")},
			wantState: StateNormal,
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.state != StateNormal {
					t.Errorf("state = %d, want StateNormal", m.state)
				}
			},
		},
		{
			name: "cost update no crash",
			msg:  agent.CostUpdateMsg{Cost: 0.1234},
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				// Just verify no panic occurred
			},
		},
		{
			name: "approval request sets awaiting approval",
			msg: ApprovalRequestMsg{
				Command:        "rm -rf /",
				Classification: "risky",
				ResponseCh:     make(chan bool, 1),
			},
			wantState: StateAwaitingApproval,
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.state != StateAwaitingApproval {
					t.Errorf("state = %d, want StateAwaitingApproval", m.state)
				}
				if m.pendingApproval == nil {
					t.Error("pendingApproval should not be nil")
				}
			},
		},
		{
			name: "approval key y approves",
			setup: func(m *Model) {
				m.state = StateAwaitingApproval
				m.pendingApproval = &ApprovalRequestMsg{
					Command:        "ls",
					Classification: "safe",
					ResponseCh:     make(chan bool, 1),
				}
			},
			msg:       tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")},
			wantState: StateStreaming,
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.state != StateStreaming {
					t.Errorf("state = %d, want StateStreaming", m.state)
				}
				if m.pendingApproval != nil {
					t.Error("pendingApproval should be nil after approval")
				}
			},
		},
		{
			name: "approval key n denies",
			setup: func(m *Model) {
				m.state = StateAwaitingApproval
				m.pendingApproval = &ApprovalRequestMsg{
					Command:        "rm -rf /",
					Classification: "risky",
					ResponseCh:     make(chan bool, 1),
				}
			},
			msg:       tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")},
			wantState: StateStreaming,
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.state != StateStreaming {
					t.Errorf("state = %d, want StateStreaming", m.state)
				}
				if m.pendingApproval != nil {
					t.Error("pendingApproval should be nil after denial")
				}
			},
		},
		{
			name: "ctrl+c quits",
			msg:  tea.KeyMsg{Type: tea.KeyCtrlC},
			checkFn: func(t *testing.T, _ Model, cmd tea.Cmd) {
				if cmd == nil {
					t.Fatal("expected a quit command, got nil")
				}
				msg := cmd()
				if _, ok := msg.(tea.QuitMsg); !ok {
					t.Errorf("expected tea.QuitMsg, got %T", msg)
				}
			},
		},
		{
			name: "exit msg quits",
			msg:  ExitMsg{},
			checkFn: func(t *testing.T, _ Model, cmd tea.Cmd) {
				if cmd == nil {
					t.Fatal("expected a quit command, got nil")
				}
				msg := cmd()
				if _, ok := msg.(tea.QuitMsg); !ok {
					t.Errorf("expected tea.QuitMsg, got %T", msg)
				}
			},
		},
		{
			name: "system msg appends to viewport",
			msg:  SystemMsg{Content: "hello system"},
			checkFn: func(t *testing.T, _ Model, _ tea.Cmd) {
				// Just verify no panic
			},
		},
		{
			name: "skill activate with nil manager shows error",
			msg:  SkillActivateMsg{Name: "test"},
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.state != StateNormal {
					t.Errorf("state = %d, want StateNormal", m.state)
				}
			},
		},
		{
			name: "wizard cancelled returns to normal",
			setup: func(m *Model) {
				m.state = StateWizard
				m.showWizard = true
				m.activeSkill = nil
			},
			msg:       WizardCancelledMsg{},
			wantState: StateNormal,
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.showWizard {
					t.Error("showWizard should be false")
				}
				if m.activeSkill != nil {
					t.Error("activeSkill should be nil")
				}
			},
		},
		{
			name: "wizard complete returns to normal",
			setup: func(m *Model) {
				m.state = StateWizard
				m.showWizard = true
				m.activeSkill = nil
			},
			msg:       WizardCompleteMsg{Values: map[string]string{"name": "val"}},
			wantState: StateNormal,
			checkFn: func(t *testing.T, m Model, _ tea.Cmd) {
				if m.showWizard {
					t.Error("showWizard should be false")
				}
			},
		},
		{
			name: "submit slash command routes through router",
			setup: func(m *Model) {
				m.router = NewSlashRouter()
			},
			msg: SubmitMsg{Content: "/help"},
			checkFn: func(t *testing.T, m Model, cmd tea.Cmd) {
				if cmd == nil {
					t.Fatal("expected a command, got nil")
				}
				msg := cmd()
				if _, ok := msg.(SystemMsg); !ok {
					t.Errorf("expected SystemMsg, got %T", msg)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel(nil, nil, nil)
			if tt.setup != nil {
				tt.setup(&m)
			}

			result, cmd := m.Update(tt.msg)
			updated := result.(Model)

			if tt.checkFn != nil {
				tt.checkFn(t, updated, cmd)
			}
		})
	}
}

func TestLoadHistory(t *testing.T) {
	m := NewModel(nil, nil, nil)
	entries := []HistoryEntry{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}

	// Just verify no panic
	m.LoadHistory(entries)
}

func TestModel_MCPApprovalRequest(t *testing.T) {
	m := NewModel(nil, nil, nil)
	ch := make(chan bool, 1)
	msg := MCPApprovalRequestMsg{
		ServerName:  "test",
		ToolName:    "tool",
		Description: "desc",
		Args:        `{"x":1}`,
		ResponseCh:  ch,
	}
	result, _ := m.Update(msg)
	updated := result.(Model)
	if updated.state != StateMcpApproval {
		t.Errorf("state = %d, want StateMcpApproval", updated.state)
	}
	if updated.pendingMCPApproval == nil {
		t.Fatal("pendingMCPApproval should be set")
	}
	if !updated.showApprovalDialog {
		t.Error("showApprovalDialog should be true")
	}
}

func TestModel_MCPApprovalKeyY(t *testing.T) {
	m := NewModel(nil, nil, nil)
	ch := make(chan bool, 1)
	m.state = StateMcpApproval
	m.pendingMCPApproval = &MCPApprovalRequestMsg{
		ServerName: "test",
		ToolName:   "tool",
		ResponseCh: ch,
	}
	m.approvalDialog = NewApprovalDialog("test", "tool", "", "", ch)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	updated := result.(Model)
	if updated.state != StateStreaming {
		t.Errorf("state = %d, want StateStreaming", updated.state)
	}
	select {
	case v := <-ch:
		if !v {
			t.Error("expected true on channel")
		}
	default:
		t.Error("expected value on channel")
	}
}

func TestModel_MCPApprovalKeyN(t *testing.T) {
	m := NewModel(nil, nil, nil)
	ch := make(chan bool, 1)
	m.state = StateMcpApproval
	m.pendingMCPApproval = &MCPApprovalRequestMsg{
		ServerName: "test",
		ToolName:   "tool",
		ResponseCh: ch,
	}
	m.approvalDialog = NewApprovalDialog("test", "tool", "", "", ch)
	result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updated := result.(Model)
	if updated.state != StateStreaming {
		t.Errorf("state = %d, want StateStreaming", updated.state)
	}
	select {
	case v := <-ch:
		if v {
			t.Error("expected false on channel")
		}
	default:
		t.Error("expected value on channel")
	}
}

func TestModel_McpErrorMsg(t *testing.T) {
	m := NewModel(nil, nil, nil)
	result, _ := m.Update(McpErrorMsg{ServerName: "srv", Error: "fail"})
	updated := result.(Model)
	if updated.state != StateNormal {
		t.Errorf("state = %d, want StateNormal", updated.state)
	}
}

func TestModel_ResourceBrowserOpen(t *testing.T) {
	m := NewModel(nil, nil, nil)
	result, _ := m.Update(ResourceBrowserOpenMsg{})
	updated := result.(Model)
	if updated.state != StateResourceBrowser {
		t.Errorf("state = %d, want StateResourceBrowser", updated.state)
	}
	if !updated.showResourceBrowser {
		t.Error("expected showResourceBrowser true")
	}
}

func TestModel_ResourceListLoaded(t *testing.T) {
	m := NewModel(nil, nil, nil)
	m.state = StateResourceBrowser
	items := []ResourceItem{{ServerName: "s", URI: "u", Name: "n"}}
	result, _ := m.Update(ResourceListLoadedMsg{Items: items})
	updated := result.(Model)
	if len(updated.resourceBrowser.items) != 1 {
		t.Errorf("items = %d, want 1", len(updated.resourceBrowser.items))
	}
}

func TestModel_ResourceAttachDetach(t *testing.T) {
	m := NewModel(nil, nil, nil)
	result, _ := m.Update(ResourceAttachMsg{Resource: mcp.PinnedResource{ServerName: "s", URI: "u", Name: "n"}})
	updated := result.(Model)
	if updated.state != StateNormal {
		t.Errorf("state = %d, want StateNormal", updated.state)
	}

	result2, _ := updated.Update(ResourceDetachMsg{ServerName: "s", URI: "u"})
	updated2 := result2.(Model)
	if updated2.state != StateNormal {
		t.Errorf("state = %d, want StateNormal", updated2.state)
	}
}

func TestModel_McpServerListNoManager(t *testing.T) {
	m := NewModel(nil, nil, nil)
	result, _ := m.Update(McpServerListMsg{})
	updated := result.(Model)
	if updated.state != StateNormal {
		t.Errorf("state = %d, want StateNormal", updated.state)
	}
}
