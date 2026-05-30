package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/omnicli/omnicli/internal/skills"
	"github.com/omnicli/omnicli/internal/tools"
)

func TestSubAgentRun_Success(t *testing.T) {
	client := &mockLLMClient{
		responses: []*ChatMessage{
			{Role: "assistant", Content: "Done"},
		},
		usages: []*Usage{{Cost: 0.5}},
	}

	registry := tools.NewRegistry()
	registry.Register(&mockTool{name: "grep_search", result: "found"})

	skillDir := t.TempDir()
	skillManager := skills.NewManager("")
	skillManager.SetGlobalDir(skillDir)
	_ = os.WriteFile(filepath.Join(skillDir, "test-skill.json"), []byte(`{
		"name": "test-skill",
		"system_prompt": "You are a test skill.",
		"tools": ["grep_search"]
	}`), 0644)

	c := &collector{}
	sa := NewSubAgent("test-skill", "find TODOs", nil, client, skillManager, registry, c.send)
	result := sa.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Summary != "Done" {
		t.Errorf("summary = %q, want Done", result.Summary)
	}
	if result.TotalCost != 0.5 {
		t.Errorf("totalCost = %f, want 0.5", result.TotalCost)
	}

	sent := c.get()
	if len(sent) == 0 {
		t.Fatal("expected log messages")
	}
	startFound := false
	for _, msg := range sent {
		if log, ok := msg.(SubAgentLogMsg); ok && log.Activity == "start" {
			startFound = true
		}
	}
	if !startFound {
		t.Error("expected start log message")
	}
}

func TestSubAgentRun_ToolLoop(t *testing.T) {
	client := &mockLLMClient{
		responses: []*ChatMessage{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{ID: "1", Name: "grep_search", Arguments: `{"pattern":"TODO"}`},
				},
			},
			{Role: "assistant", Content: "Found 3 TODOs"},
		},
		usages: []*Usage{{Cost: 0.3}, {Cost: 0.4}},
	}

	registry := tools.NewRegistry()
	registry.Register(&mockTool{name: "grep_search", result: "file.go: // TODO"})

	skillDir := t.TempDir()
	skillManager := skills.NewManager("")
	skillManager.SetGlobalDir(skillDir)
	_ = os.WriteFile(filepath.Join(skillDir, "test-skill.json"), []byte(`{
		"name": "test-skill",
		"system_prompt": "You are a test skill.",
		"tools": ["grep_search"]
	}`), 0644)

	c := &collector{}
	sa := NewSubAgent("test-skill", "find TODOs", nil, client, skillManager, registry, c.send)
	result := sa.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}
	if result.Summary != "Found 3 TODOs" {
		t.Errorf("summary = %q, want 'Found 3 TODOs'", result.Summary)
	}
	if result.TotalCost != 0.7 {
		t.Errorf("totalCost = %f, want 0.7", result.TotalCost)
	}
	if len(result.ToolsCalled) != 1 || result.ToolsCalled[0] != "grep_search" {
		t.Errorf("toolsCalled = %v, want [grep_search]", result.ToolsCalled)
	}
}

func TestSubAgentRun_CostCeiling(t *testing.T) {
	client := &mockLLMClient{
		responses: []*ChatMessage{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{ID: "1", Name: "grep_search", Arguments: `{}`},
				},
			},
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{ID: "2", Name: "grep_search", Arguments: `{}`},
				},
			},
			{Role: "assistant", Content: "should not reach"},
		},
		usages: []*Usage{{Cost: 3.0}, {Cost: 3.0}, {Cost: 0.1}},
	}

	registry := tools.NewRegistry()
	registry.Register(&mockTool{name: "grep_search", result: "found"})

	skillDir := t.TempDir()
	skillManager := skills.NewManager("")
	skillManager.SetGlobalDir(skillDir)
	_ = os.WriteFile(filepath.Join(skillDir, "test-skill.json"), []byte(`{
		"name": "test-skill",
		"system_prompt": "You are a test skill.",
		"tools": ["grep_search"]
	}`), 0644)

	c := &collector{}
	sa := NewSubAgent("test-skill", "do something", nil, client, skillManager, registry, c.send)
	result := sa.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure due to cost ceiling")
	}
	if result.Error != "cost ceiling reached at $6.00" {
		t.Errorf("error = %q, want 'cost ceiling reached at $6.00'", result.Error)
	}
	if result.TotalCost != 6.0 {
		t.Errorf("totalCost = %f, want 6.0", result.TotalCost)
	}
}

func TestSubAgentRun_ContextCancellation(t *testing.T) {
	client := &mockLLMClient{
		responses: []*ChatMessage{
			{Role: "assistant", Content: "first"},
		},
		usages: []*Usage{{Cost: 0.1}},
	}

	registry := tools.NewRegistry()

	skillDir := t.TempDir()
	skillManager := skills.NewManager("")
	skillManager.SetGlobalDir(skillDir)
	_ = os.WriteFile(filepath.Join(skillDir, "test-skill.json"), []byte(`{
		"name": "test-skill",
		"system_prompt": "You are a test skill.",
		"tools": []
	}`), 0644)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sa := NewSubAgent("test-skill", "do something", nil, client, skillManager, registry, nil)
	result := sa.Run(ctx)

	if result.Success {
		t.Fatal("expected failure due to cancellation")
	}
	if result.Error != "cancelled" {
		t.Errorf("error = %q, want 'cancelled'", result.Error)
	}
}

func TestSubAgentRun_ContextFiles(t *testing.T) {
	client := &mockLLMClient{
		responses: []*ChatMessage{
			{Role: "assistant", Content: "Done"},
		},
		usages: []*Usage{{Cost: 0.1}},
	}

	registry := tools.NewRegistry()

	skillDir := t.TempDir()
	skillManager := skills.NewManager("")
	skillManager.SetGlobalDir(skillDir)
	_ = os.WriteFile(filepath.Join(skillDir, "test-skill.json"), []byte(`{
		"name": "test-skill",
		"system_prompt": "You are a test skill.",
		"tools": []
	}`), 0644)

	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "context.txt")
	_ = os.WriteFile(validFile, []byte("hello world"), 0644)
	missingFile := filepath.Join(tmpDir, "missing.txt")

	c := &collector{}
	sa := NewSubAgent("test-skill", "read context", []string{validFile, missingFile}, client, skillManager, registry, c.send)
	result := sa.Run(context.Background())

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// Verify that a warning was sent for the missing file.
	sent := c.get()
	hasError := false
	for _, msg := range sent {
		if log, ok := msg.(SubAgentLogMsg); ok && log.Activity == "error" {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Error("expected error log for missing context file")
	}
}

func TestSubAgentRun_RecursionGuard(t *testing.T) {
	// Generate 10 identical responses so the loop runs to max iterations.
	responses := make([]*ChatMessage, 10)
	usages := make([]*Usage, 10)
	for i := 0; i < 10; i++ {
		responses[i] = &ChatMessage{
			Role: "assistant",
			ToolCalls: []ToolCall{
				{ID: fmt.Sprintf("%d", i+1), Name: "spawn_subagent", Arguments: `{}`},
			},
		}
		usages[i] = &Usage{Cost: 0.1}
	}

	client := &mockLLMClient{
		responses: responses,
		usages:    usages,
	}

	registry := tools.NewRegistry()
	registry.Register(&mockTool{name: "spawn_subagent", result: "should not be called"})

	skillDir := t.TempDir()
	skillManager := skills.NewManager("")
	skillManager.SetGlobalDir(skillDir)
	_ = os.WriteFile(filepath.Join(skillDir, "test-skill.json"), []byte(`{
		"name": "test-skill",
		"system_prompt": "You are a test skill.",
		"tools": ["spawn_subagent"]
	}`), 0644)

	sa := NewSubAgent("test-skill", "spawn another", nil, client, skillManager, registry, nil)
	result := sa.Run(context.Background())

	// The filtered registry should not contain spawn_subagent, so the tool call returns an error.
	if result.Success {
		t.Fatal("expected failure because spawn_subagent is excluded")
	}
	if result.Error != "max iterations exceeded (10)" {
		// The LLM keeps trying spawn_subagent because it's not in the registry,
		// and the error result is fed back. After max iterations it fails.
		t.Errorf("error = %q, want 'max iterations exceeded (10)'", result.Error)
	}
}

func TestSubAgentRun_InvalidSkill(t *testing.T) {
	client := &mockLLMClient{}
	registry := tools.NewRegistry()
	skillManager := skills.NewManager("")

	sa := NewSubAgent("nonexistent", "do something", nil, client, skillManager, registry, nil)
	result := sa.Run(context.Background())

	if result.Success {
		t.Fatal("expected failure")
	}
	if result.Error != "skill not found: nonexistent" {
		t.Errorf("error = %q, want 'skill not found: nonexistent'", result.Error)
	}
}
