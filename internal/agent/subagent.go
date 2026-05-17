package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/omnicli/omnicli/internal/skills"
	"github.com/omnicli/omnicli/internal/tools"
)

// SubAgentLogMsg is sent during sub-agent execution to report activity.
type SubAgentLogMsg struct {
	SkillName string
	Activity  string // "start", "tool_call", "response", "done", "error"
	Detail    string
}

// SubAgentResult contains the outcome of a sub-agent run.
type SubAgentResult struct {
	Summary     string   `json:"summary"`
	FullOutput  string   `json:"full_output"`
	ToolsCalled []string `json:"tools_called"`
	TotalCost   float64  `json:"total_cost"`
	Success     bool     `json:"success"`
	Error       string   `json:"error,omitempty"`
}

// SubAgent executes a delegated task in an isolated session with a cost ceiling.
type SubAgent struct {
	skillName     string
	prompt        string
	contextFiles  []string
	client        LLMClient
	skills        *skills.Manager
	tools         *tools.Registry
	send          SendFunc
	costLimit     float64
	maxIterations int
}

// NewSubAgent creates a SubAgent with default cost limit and iteration cap.
func NewSubAgent(skillName, prompt string, contextFiles []string, client LLMClient, skills *skills.Manager, tools *tools.Registry, send SendFunc) *SubAgent {
	return &SubAgent{
		skillName:     skillName,
		prompt:        prompt,
		contextFiles:  contextFiles,
		client:        client,
		skills:        skills,
		tools:         tools,
		send:          send,
		costLimit:     5.0,
		maxIterations: 10,
	}
}

// Run executes the sub-agent loop until completion, cost ceiling, or error.
func (sa *SubAgent) Run(ctx context.Context) SubAgentResult {
	if sa.send != nil {
		sa.send(SubAgentLogMsg{SkillName: sa.skillName, Activity: "start", Detail: "sub-agent started"})
	}

	skill, err := sa.skills.Get(sa.skillName)
	if err != nil {
		return sa.finish(false, fmt.Sprintf("skill not found: %s", sa.skillName), 0)
	}

	filtered := sa.tools.Filter(skill.Tools)
	toolDefs := buildToolDefs(filtered)

	messages := []ChatMessage{
		{Role: "user", Content: sa.buildInitialInput()},
	}

	var totalCost float64
	var toolsCalled []string
	var fullOutput strings.Builder

	for i := 0; i < sa.maxIterations; i++ {
		if totalCost >= sa.costLimit {
			return sa.finish(false, fmt.Sprintf("cost ceiling reached at $%.2f", totalCost), totalCost)
		}

		finalMsg, usage, err := sa.client.ChatStream(ctx, messages, toolDefs, func(chunk StreamChunk) {
			if chunk.Content != "" {
				fullOutput.WriteString(chunk.Content)
				if sa.send != nil {
					sa.send(SubAgentLogMsg{SkillName: sa.skillName, Activity: "response", Detail: chunk.Content})
				}
			}
		})
		if err != nil {
			if ctx.Err() != nil {
				return sa.finish(false, "cancelled", totalCost)
			}
			return sa.finish(false, fmt.Sprintf("LLM request failed: %v", err), totalCost)
		}

		if usage != nil {
			totalCost += usage.Cost
		}

		if len(finalMsg.ToolCalls) == 0 {
			return SubAgentResult{
				Success:     true,
				Summary:     finalMsg.Content,
				FullOutput:  fullOutput.String(),
				ToolsCalled: toolsCalled,
				TotalCost:   totalCost,
			}
		}

		messages = append(messages, *finalMsg)

		for _, tc := range finalMsg.ToolCalls {
			toolsCalled = append(toolsCalled, tc.Name)
			if sa.send != nil {
				sa.send(SubAgentLogMsg{SkillName: sa.skillName, Activity: "tool_call", Detail: fmt.Sprintf("%s(%s)", tc.Name, tc.Arguments)})
			}
			result := sa.executeTool(ctx, filtered, tc)
			messages = append(messages, ChatMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return sa.finish(false, fmt.Sprintf("max iterations exceeded (%d)", sa.maxIterations), totalCost)
}

func (sa *SubAgent) buildInitialInput() string {
	var b strings.Builder
	b.WriteString(sa.prompt)

	for _, path := range sa.contextFiles {
		data, err := os.ReadFile(path)
		if err != nil {
			if sa.send != nil {
				sa.send(SubAgentLogMsg{SkillName: sa.skillName, Activity: "error", Detail: fmt.Sprintf("failed to read context file %s: %v", path, err)})
			}
			continue
		}
		b.WriteString(fmt.Sprintf("\n\n--- %s ---\n%s", path, string(data)))
	}

	return b.String()
}

func (sa *SubAgent) executeTool(ctx context.Context, registry *tools.Registry, tc ToolCall) string {
	tool, ok := registry.Get(tc.Name)
	if !ok {
		return fmt.Sprintf("error: tool %q not found", tc.Name)
	}
	result, err := tool.Execute(ctx, json.RawMessage(tc.Arguments))
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return result
}

func (sa *SubAgent) finish(success bool, errMsg string, cost float64) SubAgentResult {
	if errMsg != "" && sa.send != nil {
		sa.send(SubAgentLogMsg{SkillName: sa.skillName, Activity: "error", Detail: errMsg})
	}
	return SubAgentResult{
		Success:   success,
		Error:     errMsg,
		TotalCost: cost,
	}
}

func buildToolDefs(registry *tools.Registry) []ToolDefinition {
	registered := registry.List()
	defs := make([]ToolDefinition, 0, len(registered))
	for _, t := range registered {
		defs = append(defs, ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}
	return defs
}
