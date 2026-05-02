package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSlashRouter(t *testing.T) {
	r := NewSlashRouter()

	cmds := r.List()
	if len(cmds) == 0 {
		t.Fatal("expected built-in commands to be registered")
	}

	// Should have at least exit and help.
	found := make(map[string]bool)
	for _, c := range cmds {
		found[c] = true
	}
	if !found["exit"] {
		t.Error("expected /exit to be registered")
	}
	if !found["help"] {
		t.Error("expected /help to be registered")
	}
	if !found["skills"] {
		t.Error("expected /skills to be registered")
	}
	if !found["skill"] {
		t.Error("expected /skill to be registered")
	}
}

func TestSlashRouterDispatch(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantHandled bool
		wantMsgType string
	}{
		{
			name:        "non-slash input",
			input:       "hello world",
			wantHandled: false,
		},
		{
			name:        "empty slash",
			input:       "/",
			wantHandled: true,
			wantMsgType: "SystemMsg",
		},
		{
			name:        "exit command",
			input:       "/exit",
			wantHandled: true,
			wantMsgType: "ExitMsg",
		},
		{
			name:        "help command",
			input:       "/help",
			wantHandled: true,
			wantMsgType: "SystemMsg",
		},
		{
			name:        "skill command with args",
			input:       "/skill docs-expert",
			wantHandled: true,
			wantMsgType: "SkillActivateMsg",
		},
		{
			name:        "skill command without args",
			input:       "/skill",
			wantHandled: true,
			wantMsgType: "SystemMsg",
		},
		{
			name:        "skill command with extra args",
			input:       "/skill docs-expert --force",
			wantHandled: true,
			wantMsgType: "SkillActivateMsg",
		},
		{
			name:        "unknown command",
			input:       "/foo",
			wantHandled: true,
			wantMsgType: "SystemMsg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewSlashRouter()
			handled, msg, _ := r.Dispatch(tt.input)

			if handled != tt.wantHandled {
				t.Errorf("handled = %v, want %v", handled, tt.wantHandled)
			}

			if tt.wantMsgType != "" {
				if msg == nil {
					t.Fatalf("expected msg of type %s, got nil", tt.wantMsgType)
				}
				var gotType string
				switch msg.(type) {
				case ExitMsg:
					gotType = "ExitMsg"
				case SystemMsg:
					gotType = "SystemMsg"
				case SkillActivateMsg:
					gotType = "SkillActivateMsg"
				default:
					gotType = "other"
				}
				if gotType != tt.wantMsgType {
					t.Errorf("msg type = %s, want %s", gotType, tt.wantMsgType)
				}
			}
		})
	}
}

func TestSlashRouterRegister(t *testing.T) {
	r := NewSlashRouter()

	customCalled := false
	var customArgs string
	r.Register("custom", func(args string) (tea.Msg, tea.Cmd) {
		customCalled = true
		customArgs = args
		return SystemMsg{Content: "custom"}, nil
	})

	handled, msg, _ := r.Dispatch("/custom hello")
	if !handled {
		t.Error("expected custom command to be handled")
	}
	if !customCalled {
		t.Error("expected custom handler to be called")
	}
	if customArgs != "hello" {
		t.Errorf("custom args = %q, want %q", customArgs, "hello")
	}
	if msg == nil {
		t.Error("expected non-nil msg")
	}
}

func TestSlashRouterRegisterOverwrite(t *testing.T) {
	r := NewSlashRouter()

	r.Register("exit", func(args string) (tea.Msg, tea.Cmd) {
		return SystemMsg{Content: "overridden"}, nil
	})

	handled, msg, _ := r.Dispatch("/exit")
	if !handled {
		t.Fatal("expected handled")
	}
	sys, ok := msg.(SystemMsg)
	if !ok {
		t.Fatalf("expected SystemMsg, got %T", msg)
	}
	if sys.Content != "overridden" {
		t.Errorf("content = %q, want %q", sys.Content, "overridden")
	}
}

func TestSlashRouterList(t *testing.T) {
	r := NewSlashRouter()

	cmds := r.List()
	if len(cmds) == 0 {
		t.Fatal("expected non-empty command list")
	}

	for i := 1; i < len(cmds); i++ {
		if cmds[i] < cmds[i-1] {
			t.Error("expected command list to be sorted")
		}
	}
}

func TestSlashRouterDescriptions(t *testing.T) {
	r := NewSlashRouter()

	descs := r.Descriptions()
	if len(descs) == 0 {
		t.Fatal("expected non-empty descriptions")
	}

	for _, d := range descs {
		if d.Name == "" {
			t.Error("expected non-empty command name")
		}
		if d.Description == "" {
			t.Errorf("expected non-empty description for command %q", d.Name)
		}
	}

	// All registered handlers should have descriptions.
	cmds := r.List()
	if len(descs) != len(cmds) {
		t.Errorf("descriptions count = %d, want %d", len(descs), len(cmds))
	}
}
