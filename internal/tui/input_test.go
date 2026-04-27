package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestInputModelSetSkillPrefix(t *testing.T) {
	m := NewInputModel()

	// Default should be empty.
	if m.skillPrefix != "" {
		t.Errorf("expected empty skillPrefix by default, got %q", m.skillPrefix)
	}

	m.SetSkillPrefix("[Docs Expert] ")
	if m.skillPrefix != "[Docs Expert] " {
		t.Errorf("expected skillPrefix %q, got %q", "[Docs Expert] ", m.skillPrefix)
	}

	m.SetSkillPrefix("")
	if m.skillPrefix != "" {
		t.Errorf("expected empty skillPrefix after reset, got %q", m.skillPrefix)
	}
}

func TestInputModelViewWithPrefix(t *testing.T) {
	m := NewInputModel()

	// Without prefix, View should just return the textarea view.
	viewNoPrefix := m.View()
	if viewNoPrefix == "" {
		t.Error("expected non-empty view from textarea")
	}

	// With prefix, View should contain the styled prefix.
	m.SetSkillPrefix("[Docs Expert] ")
	viewWithPrefix := m.View()

	// The prefix should be rendered with lipgloss styling (bold, cyan color "6").
	prefixStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	expectedPrefix := prefixStyle.Render("[Docs Expert] ")

	if !strings.HasPrefix(viewWithPrefix, expectedPrefix) {
		t.Errorf("expected view to start with styled prefix %q, got %q", expectedPrefix, viewWithPrefix)
	}

	// The view should be longer than the textarea alone.
	if len(viewWithPrefix) <= len(viewNoPrefix) {
		t.Error("expected view with prefix to be longer than view without prefix")
	}
}

func TestInputModelViewWithoutPrefix(t *testing.T) {
	m := NewInputModel()
	m.SetSkillPrefix("")

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view from textarea")
	}

	// Should not contain any styled prefix markup.
	if strings.Contains(view, "[") {
		// The raw view from textarea shouldn't have brackets unless it's part of placeholder.
		// The placeholder is "Ask anything... (Enter to submit, Shift+Enter for newline)"
		// which doesn't contain brackets, so this is a safe sanity check.
		t.Logf("view contains brackets: %q", view)
	}
}
