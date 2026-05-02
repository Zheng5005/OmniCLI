package tui

import (
	"strings"
	"testing"
)

func TestStatusBarModel_ViewWithSpinnerPrefix(t *testing.T) {
	sb := NewStatusBarModel("gpt-4", 80)
	sb.SetActivity(ActivityThinking)

	view := sb.View("⠋")
	if !strings.Contains(view, "⠋") {
		t.Error("expected spinner frame in view")
	}
	if !strings.Contains(view, "Thinking") {
		t.Error("expected activity label 'Thinking' in view")
	}

	// Spinner should appear before activity text
	spinnerIdx := strings.Index(view, "⠋")
	thinkingIdx := strings.Index(view, "Thinking")
	if spinnerIdx == -1 || thinkingIdx == -1 || spinnerIdx > thinkingIdx {
		t.Error("expected spinner to appear before activity text")
	}
}

func TestStatusBarModel_ViewWithoutSpinner(t *testing.T) {
	sb := NewStatusBarModel("gpt-4", 80)
	sb.SetActivity(ActivityReady)

	view := sb.View("")
	if strings.Contains(view, "⠋") {
		t.Error("unexpected spinner frame in ready state view")
	}
	if !strings.Contains(view, "Ready") {
		t.Error("expected 'Ready' in view")
	}
}

func TestStatusBarModel_ViewBoldForActiveStates(t *testing.T) {
	activeStates := []Activity{ActivityThinking, ActivitySearching, ActivityExecuting}
	for _, activity := range activeStates {
		sb := NewStatusBarModel("gpt-4", 80)
		sb.SetActivity(activity)
		view := sb.View("x")
		if view == "" {
			t.Errorf("expected non-empty view for activity %v", activity)
		}
	}
}
