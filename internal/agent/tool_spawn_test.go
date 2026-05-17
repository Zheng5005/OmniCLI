package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/omnicli/omnicli/internal/skills"
	"github.com/omnicli/omnicli/internal/tools"
)

type mockClientFactory struct {
	client LLMClient
	err    error
}

func (f *mockClientFactory) create(systemPrompt string, toolNames []string) (LLMClient, error) {
	return f.client, f.err
}

func TestSpawnSubAgentTool_Execute_ValidParams(t *testing.T) {
	mockClient := &mockLLMClient{
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

	tool := &SpawnSubAgentTool{
		factory: &mockClientFactory{client: mockClient},
		skills:  skillManager,
		tools:   registry,
		send:    nil,
	}

	args := json.RawMessage(`{"skill_name":"test-skill","sub_task_prompt":"find TODOs","context_files":["file.go"]}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	var sar SubAgentResult
	if err := json.Unmarshal([]byte(result), &sar); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if !sar.Success {
		t.Fatalf("expected success, got error: %s", sar.Error)
	}
	if sar.Summary != "Done" {
		t.Errorf("summary = %q, want Done", sar.Summary)
	}
	if sar.TotalCost != 0.5 {
		t.Errorf("totalCost = %f, want 0.5", sar.TotalCost)
	}
}

func TestSpawnSubAgentTool_Execute_InvalidSkill(t *testing.T) {
	registry := tools.NewRegistry()
	skillManager := skills.NewManager("")

	tool := &SpawnSubAgentTool{
		factory: &mockClientFactory{},
		skills:  skillManager,
		tools:   registry,
		send:    nil,
	}

	args := json.RawMessage(`{"skill_name":"missing","sub_task_prompt":"test"}`)
	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error for missing skill")
	}
	if err.Error() != "skill not found: missing" {
		t.Errorf("error = %q, want 'skill not found: missing'", err.Error())
	}
}

func TestSpawnSubAgentTool_Execute_MissingParams(t *testing.T) {
	registry := tools.NewRegistry()
	skillManager := skills.NewManager("")

	tool := &SpawnSubAgentTool{
		factory: &mockClientFactory{},
		skills:  skillManager,
		tools:   registry,
		send:    nil,
	}

	tests := []struct {
		name string
		args string
		want string
	}{
		{
			name: "missing skill_name",
			args: `{"sub_task_prompt":"test"}`,
			want: "skill_name is required",
		},
		{
			name: "missing sub_task_prompt",
			args: `{"skill_name":"test"}`,
			want: "sub_task_prompt is required",
		},
		{
			name: "empty args",
			args: `{}`,
			want: "skill_name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), json.RawMessage(tt.args))
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.want {
				t.Errorf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestSpawnSubAgentTool_Execute_ClientFactoryError(t *testing.T) {
	registry := tools.NewRegistry()
	skillDir := t.TempDir()
	skillManager := skills.NewManager("")
	skillManager.SetGlobalDir(skillDir)
	_ = os.WriteFile(filepath.Join(skillDir, "test-skill.json"), []byte(`{
		"name": "test-skill",
		"system_prompt": "You are a test skill.",
		"tools": []
	}`), 0644)

	tool := &SpawnSubAgentTool{
		factory: &mockClientFactory{err: fmt.Errorf("factory error")},
		skills:  skillManager,
		tools:   registry,
		send:    nil,
	}

	args := json.RawMessage(`{"skill_name":"test-skill","sub_task_prompt":"test"}`)
	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "failed to create sub-agent LLM client: factory error" {
		t.Errorf("error = %q, want 'failed to create sub-agent LLM client: factory error'", err.Error())
	}
}

func TestSpawnSubAgentTool_Interface(t *testing.T) {
	tool := &SpawnSubAgentTool{}

	if tool.Name() != "spawn_subagent" {
		t.Errorf("name = %q, want spawn_subagent", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}

	params := tool.Parameters()
	if len(params) == 0 {
		t.Error("expected non-empty parameters")
	}
}
