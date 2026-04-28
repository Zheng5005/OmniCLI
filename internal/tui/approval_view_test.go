package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestApprovalDialog_Init(t *testing.T) {
	d := NewApprovalDialog("srv", "tool", "desc", `{"x":1}`, make(chan bool, 1))
	if d.Init() != nil {
		t.Error("expected nil init cmd")
	}
}

func TestApprovalDialog_InitialApproved(t *testing.T) {
	d := NewApprovalDialog("srv", "tool", "desc", `{"x":1}`, make(chan bool, 1))
	if d.Approved() != nil {
		t.Error("expected nil before decision")
	}
}

func TestApprovalDialog_EnterApproves(t *testing.T) {
	ch := make(chan bool, 1)
	d := NewApprovalDialog("srv", "tool", "desc", `{"x":1}`, ch)
	updated, _ := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	ud := updated.(ApprovalDialog)
	if ud.Approved() == nil || !*ud.Approved() {
		t.Error("expected approved true")
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

func TestApprovalDialog_TabSelectsDeny(t *testing.T) {
	ch := make(chan bool, 1)
	d := NewApprovalDialog("srv", "tool", "desc", `{"x":1}`, ch)
	updated, _ := d.Update(tea.KeyMsg{Type: tea.KeyTab})
	ud := updated.(ApprovalDialog)
	updated2, _ := ud.Update(tea.KeyMsg{Type: tea.KeyEnter})
	ud2 := updated2.(ApprovalDialog)
	if ud2.Approved() == nil || *ud2.Approved() {
		t.Error("expected approved false")
	}
}

func TestApprovalDialog_YKeyApproves(t *testing.T) {
	ch := make(chan bool, 1)
	d := NewApprovalDialog("srv", "tool", "desc", `{"x":1}`, ch)
	updated, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	ud := updated.(ApprovalDialog)
	if ud.Approved() == nil || !*ud.Approved() {
		t.Error("expected approved true")
	}
}

func TestApprovalDialog_NKeyDenies(t *testing.T) {
	ch := make(chan bool, 1)
	d := NewApprovalDialog("srv", "tool", "desc", `{"x":1}`, ch)
	updated, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	ud := updated.(ApprovalDialog)
	if ud.Approved() == nil || *ud.Approved() {
		t.Error("expected approved false")
	}
}

func TestApprovalDialog_JKeySelectsDeny(t *testing.T) {
	d := NewApprovalDialog("srv", "tool", "desc", `{"x":1}`, make(chan bool, 1))
	updated, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	ud := updated.(ApprovalDialog)
	updated2, _ := ud.Update(tea.KeyMsg{Type: tea.KeyEnter})
	ud2 := updated2.(ApprovalDialog)
	if ud2.Approved() == nil || *ud2.Approved() {
		t.Error("expected denied after j+enter")
	}
}

func TestApprovalDialog_KKeySelectsApprove(t *testing.T) {
	d := NewApprovalDialog("srv", "tool", "desc", `{"x":1}`, make(chan bool, 1))
	updated, _ := d.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	ud := updated.(ApprovalDialog)
	updated2, _ := ud.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	ud2 := updated2.(ApprovalDialog)
	updated3, _ := ud2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	ud3 := updated3.(ApprovalDialog)
	if ud3.Approved() == nil || !*ud3.Approved() {
		t.Error("expected approved after k+enter")
	}
}

func TestApprovalDialog_ViewRenders(t *testing.T) {
	d := NewApprovalDialog("srv", "tool", "desc", `{"x":1}`, make(chan bool, 1))
	d.SetSize(80, 24)
	view := d.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}
	if !strings.Contains(view, "srv") {
		t.Error("expected server name in view")
	}
	if !strings.Contains(view, "tool") {
		t.Error("expected tool name in view")
	}
	if !strings.Contains(view, "desc") {
		t.Error("expected description in view")
	}
	if !strings.Contains(view, `{"x":1}`) {
		t.Error("expected args in view")
	}
}

func TestApprovalDialog_ViewEmptyAfterApprove(t *testing.T) {
	d := NewApprovalDialog("srv", "tool", "desc", "", make(chan bool, 1))
	updated, _ := d.Update(tea.KeyMsg{Type: tea.KeyEnter})
	ud := updated.(ApprovalDialog)
	view := ud.View()
	if view != "" {
		t.Errorf("expected empty view after approval, got %q", view)
	}
}
