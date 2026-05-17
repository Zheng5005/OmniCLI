package agent

import (
	"context"

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
	return SubAgentResult{Success: false, Error: "not implemented"}
}
