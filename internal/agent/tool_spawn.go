package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Zheng5005/omnigo"
	"github.com/omnicli/omnicli/internal/skills"
	"github.com/omnicli/omnicli/internal/tools"
)

// SpawnSubAgentTool allows the main agent to delegate tasks to isolated sub-agents.
type SpawnSubAgentTool struct {
	client *omnigo.Client
	skills *skills.Manager
	tools  *tools.Registry
	send   SendFunc
}

// NewSpawnSubAgentTool creates the spawn_subagent tool.
func NewSpawnSubAgentTool(client *omnigo.Client, skills *skills.Manager, tools *tools.Registry, send SendFunc) *SpawnSubAgentTool {
	return &SpawnSubAgentTool{
		client: client,
		skills: skills,
		tools:  tools,
		send:   send,
	}
}

// Name returns the tool name.
func (t *SpawnSubAgentTool) Name() string { return "spawn_subagent" }

// Description returns what the tool does.
func (t *SpawnSubAgentTool) Description() string {
	return "Spawns an isolated sub-agent with a specific skill to handle a sub-task. The sub-agent has its own tool registry, cost ceiling, and cannot spawn further sub-agents."
}

// Parameters returns the JSON Schema for the tool's arguments.
func (t *SpawnSubAgentTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "required": ["skill_name", "sub_task_prompt"],
  "properties": {
    "skill_name": {
      "type": "string",
      "description": "Name of the skill to use for the sub-agent."
    },
    "sub_task_prompt": {
      "type": "string",
      "description": "The task prompt to send to the sub-agent."
    },
    "context_files": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Optional list of file paths to include as context."
    }
  }
}`)
}

type spawnSubAgentArgs struct {
	SkillName     string   `json:"skill_name"`
	SubTaskPrompt string   `json:"sub_task_prompt"`
	ContextFiles  []string `json:"context_files"`
}

// Execute parses parameters, validates the skill, runs the sub-agent, and returns a JSON result.
func (t *SpawnSubAgentTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var a spawnSubAgentArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}
	if a.SkillName == "" {
		return "", fmt.Errorf("skill_name is required")
	}
	if a.SubTaskPrompt == "" {
		return "", fmt.Errorf("sub_task_prompt is required")
	}

	skill, err := t.skills.Get(a.SkillName)
	if err != nil {
		return "", fmt.Errorf("skill not found: %s", a.SkillName)
	}

	llmClient, err := NewOmniGoClientFromClient(t.client, skill.SystemPrompt, skill.Tools)
	if err != nil {
		return "", fmt.Errorf("failed to create sub-agent LLM client: %w", err)
	}

	filtered := t.tools.Filter(skill.Tools)

	subAgent := NewSubAgent(a.SkillName, a.SubTaskPrompt, a.ContextFiles, llmClient, t.skills, filtered, t.send)
	result := subAgent.Run(ctx)

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal result: %w", err)
	}
	return string(resultJSON), nil
}
