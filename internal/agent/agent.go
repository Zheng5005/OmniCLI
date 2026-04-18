// Package agent orchestrates the LLM ↔ tool interaction loop for OmniCLI.
package agent

import (
	"context"
	"encoding/json"

	"github.com/omnicli/omnicli/internal/history"
	"github.com/omnicli/omnicli/internal/tools"
)

// ChatMessage represents a message in the LLM conversation.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a function call requested by the LLM.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// StreamChunk represents a piece of streaming response.
type StreamChunk struct {
	Content   string
	ToolCalls []ToolCall
	Done      bool
}

// Usage tracks token usage and cost for a single API call.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Cost             float64
}

// ToolDefinition describes a tool for the LLM API.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// LLMClient is the interface that any LLM provider must implement.
type LLMClient interface {
	// ChatStream sends messages to the LLM and streams back the response.
	// The onChunk callback is called for each chunk received.
	// Returns the complete final message, usage stats, and any error.
	ChatStream(ctx context.Context, messages []ChatMessage, toolDefs []ToolDefinition, onChunk func(StreamChunk)) (*ChatMessage, *Usage, error)

	// ModelName returns the name of the active model.
	ModelName() string
}

// SendFunc is a callback for sending messages back to the TUI.
// In production this will be tea.Program.Send.
type SendFunc func(msg interface{})

// Agent orchestrates the LLM ↔ tool interaction loop.
type Agent struct {
	client   LLMClient
	registry *tools.Registry
	session  *history.Session
	send     SendFunc
}

// New creates a new Agent with the given dependencies.
func New(client LLMClient, registry *tools.Registry, session *history.Session, send SendFunc) *Agent {
	return &Agent{
		client:   client,
		registry: registry,
		session:  session,
		send:     send,
	}
}

// ModelName returns the active model name from the LLM client.
func (a *Agent) ModelName() string {
	return a.client.ModelName()
}
