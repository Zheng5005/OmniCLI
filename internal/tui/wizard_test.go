package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewVariableWizard(t *testing.T) {
	vars := []string{"project_type", "focus_area"}
	descs := map[string]string{
		"project_type": "Type of project",
		"focus_area":   "Focus area",
	}

	w := NewVariableWizard("Docs Expert", vars, descs)

	if w.title != "Docs Expert" {
		t.Errorf("title = %q, want %q", w.title, "Docs Expert")
	}
	if len(w.inputs) != 2 {
		t.Errorf("inputs len = %d, want 2", len(w.inputs))
	}
	if w.current != 0 {
		t.Errorf("current = %d, want 0", w.current)
	}
	if w.done {
		t.Error("expected wizard to not be done initially")
	}
	if w.cancelled {
		t.Error("expected wizard to not be cancelled initially")
	}
}

func TestVariableWizardInitNoVars(t *testing.T) {
	w := NewVariableWizard("Test", nil, nil)
	cmd := w.Init()
	if cmd == nil {
		t.Fatal("expected a command for empty wizard")
	}
	msg := cmd()
	complete, ok := msg.(WizardCompleteMsg)
	if !ok {
		t.Fatalf("expected WizardCompleteMsg, got %T", msg)
	}
	if complete.Values == nil {
		t.Error("expected non-nil Values map")
	}
}

func TestVariableWizardComplete(t *testing.T) {
	vars := []string{"name", "role"}
	descs := map[string]string{
		"name": "Your name",
		"role": "Your role",
	}

	w := NewVariableWizard("Test", vars, descs)

	// Type first value and press Enter.
	w.inputs[0].SetValue("Alice")
	m, cmd := w.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard := m.(VariableWizard)

	if wizard.current != 1 {
		t.Errorf("current = %d, want 1 after first Enter", wizard.current)
	}
	if cmd == nil {
		t.Error("expected a command after moving to next variable")
	}

	// Type second value and press Enter.
	wizard.inputs[1].SetValue("Developer")
	m, cmd = wizard.Update(tea.KeyMsg{Type: tea.KeyEnter})
	wizard = m.(VariableWizard)

	if !wizard.done {
		t.Error("expected wizard to be done after last variable")
	}
	if cmd == nil {
		t.Fatal("expected a command after completion")
	}

	msg := cmd()
	if msg == nil {
		t.Fatal("expected WizardCompleteMsg, got nil")
	}

	complete, ok := msg.(WizardCompleteMsg)
	if !ok {
		t.Fatalf("expected WizardCompleteMsg, got %T", msg)
	}

	if complete.Values["name"] != "Alice" {
		t.Errorf("name = %q, want %q", complete.Values["name"], "Alice")
	}
	if complete.Values["role"] != "Developer" {
		t.Errorf("role = %q, want %q", complete.Values["role"], "Developer")
	}
}

func TestVariableWizardCancel(t *testing.T) {
	vars := []string{"name"}
	descs := map[string]string{"name": "Your name"}

	w := NewVariableWizard("Test", vars, descs)

	m, cmd := w.Update(tea.KeyMsg{Type: tea.KeyEscape})
	wizard := m.(VariableWizard)

	if !wizard.cancelled {
		t.Error("expected wizard to be cancelled")
	}
	if wizard.done {
		t.Error("expected wizard to not be done")
	}

	if cmd == nil {
		t.Fatal("expected a command after cancel")
	}

	msg := cmd()
	if _, ok := msg.(WizardCancelledMsg); !ok {
		t.Fatalf("expected WizardCancelledMsg, got %T", msg)
	}
}

func TestVariableWizardValues(t *testing.T) {
	vars := []string{"a", "b"}
	descs := map[string]string{"a": "A", "b": "B"}

	w := NewVariableWizard("Test", vars, descs)
	w.inputs[0].SetValue("1")
	w.inputs[1].SetValue("2")
	w.done = true

	vals := w.Values()
	if vals["a"] != "1" {
		t.Errorf("a = %q, want %q", vals["a"], "1")
	}
	if vals["b"] != "2" {
		t.Errorf("b = %q, want %q", vals["b"], "2")
	}
}

func TestVariableWizardSetSize(t *testing.T) {
	w := NewVariableWizard("Test", []string{"x"}, map[string]string{"x": "X"})
	w.SetSize(80, 24)

	if w.width != 80 {
		t.Errorf("width = %d, want 80", w.width)
	}
	if w.height != 24 {
		t.Errorf("height = %d, want 24", w.height)
	}
}

func TestVariableWizardViewEmptyWhenDone(t *testing.T) {
	w := NewVariableWizard("Test", []string{"x"}, map[string]string{"x": "X"})
	w.done = true

	if w.View() != "" {
		t.Error("expected empty view when done")
	}
}

func TestVariableWizardViewEmptyWhenCancelled(t *testing.T) {
	w := NewVariableWizard("Test", []string{"x"}, map[string]string{"x": "X"})
	w.cancelled = true

	if w.View() != "" {
		t.Error("expected empty view when cancelled")
	}
}
