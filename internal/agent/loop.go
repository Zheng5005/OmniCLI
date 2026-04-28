package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/omnicli/omnicli/internal/mcp"
)

// maxIterations is the maximum number of LLM round-trips to prevent
// infinite tool call loops.
const maxIterations = 20

// StreamChunkMsg carries a piece of streamed LLM content to the TUI.
type StreamChunkMsg struct {
	Content string
}

// ToolCallMsg notifies the TUI that a tool is being invoked.
type ToolCallMsg struct {
	Name string
	Args string
}

// AgentDoneMsg signals that the agent loop has finished.
type AgentDoneMsg struct {
	Content   string
	TotalCost float64
}

// CostUpdateMsg carries an updated cumulative cost to the TUI.
type CostUpdateMsg struct {
	Cost          float64
	McpDataTokens int
	McpDataCost   float64
}

// ErrorMsg carries an error from the agent loop to the TUI.
type ErrorMsg struct {
	Err error
}

// Run executes the agent loop for a single user prompt. It streams LLM
// responses, executes any requested tool calls, and re-invokes the LLM
// until no more tool calls remain or the iteration limit is reached.
func (a *Agent) Run(ctx context.Context, prompt string) {
	if a.client == nil {
		a.send(ErrorMsg{Err: fmt.Errorf("no LLM client configured: set GOOGLE_API_KEY, ANTHROPIC_API_KEY, or OPENAI_API_KEY and restart")})
		return
	}

	a.session.AddMessage("user", prompt)

	messages := a.buildMessages()
	toolDefs := a.buildToolDefs()

	var totalCost float64
	var totalMcpDataTokens int
	var totalMcpDataCost float64
	var lastInputRate float64

	for i := 0; i < maxIterations; i++ {
		var accumulated string

		finalMsg, usage, err := a.client.ChatStream(ctx, messages, toolDefs, func(chunk StreamChunk) {
			if chunk.Content != "" {
				accumulated += chunk.Content
				a.send(StreamChunkMsg{Content: chunk.Content})
			}
		})
		if err != nil {
			a.send(ErrorMsg{Err: fmt.Errorf("LLM request failed: %w", err)})
			return
		}

		if usage != nil {
			totalCost += usage.Cost
			if usage.PromptTokens > 0 {
				lastInputRate = usage.InputCost / float64(usage.PromptTokens)
			} else if usage.TotalTokens > 0 {
				lastInputRate = usage.Cost / float64(usage.TotalTokens)
			}
		}

		if len(finalMsg.ToolCalls) == 0 {
			content := finalMsg.Content
			if content == "" {
				content = accumulated
			}
			a.session.AddMessage("assistant", content)
			_ = a.session.Save()
			a.send(AgentDoneMsg{Content: content, TotalCost: totalCost})
			a.send(CostUpdateMsg{Cost: totalCost, McpDataTokens: totalMcpDataTokens, McpDataCost: totalMcpDataCost})
			return
		}

		// Append assistant message with tool calls to conversation.
		messages = append(messages, *finalMsg)

		// Execute each tool call and append results.
		for _, tc := range finalMsg.ToolCalls {
			a.send(ToolCallMsg{Name: tc.Name, Args: tc.Arguments})

			result := a.executeTool(ctx, tc)
			messages = append(messages, ChatMessage{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})

			// Track MCP data tokens for cost calculation.
			if tool, ok := a.registry.Get(tc.Name); ok {
				if _, isMCP := tool.(*mcp.MCPTool); isMCP {
					tokens := len(result) / 4
					totalMcpDataTokens += tokens
					if lastInputRate > 0 {
						mcpCost := float64(tokens) * lastInputRate
						totalMcpDataCost += mcpCost
						totalCost += mcpCost
					}
				}
			}
		}
	}

	a.send(ErrorMsg{Err: fmt.Errorf("agent loop exceeded maximum iterations (%d)", maxIterations)})
}

// executeTool looks up and runs a single tool call, returning the result string.
// This works generically for both built-in tools and MCP tools registered
// dynamically at runtime. MCP errors are surfaced as tool execution errors.
func (a *Agent) executeTool(ctx context.Context, tc ToolCall) string {
	tool, ok := a.registry.Get(tc.Name)
	if !ok {
		return fmt.Sprintf("error: tool %q not found", tc.Name)
	}

	result, err := tool.Execute(ctx, json.RawMessage(tc.Arguments))
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return result
}

// buildMessages converts the session history into ChatMessage slice.
func (a *Agent) buildMessages() []ChatMessage {
	msgs := make([]ChatMessage, len(a.session.Messages))
	for i, m := range a.session.Messages {
		msgs[i] = ChatMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return msgs
}

// buildToolDefs creates ToolDefinition slice from the registry.
func (a *Agent) buildToolDefs() []ToolDefinition {
	registered := a.registry.List()
	defs := make([]ToolDefinition, len(registered))
	for i, t := range registered {
		defs[i] = ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		}
	}
	return defs
}
