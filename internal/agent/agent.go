// Package agent orchestrates the LLM ↔ tool interaction loop for OmniCLI.
package agent

import (
	"context"
	"encoding/json"
	"fmt"

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
	client        LLMClient
	registry      *tools.Registry
	session       *history.Session
	send          SendFunc
	models        []string
	enablePricing bool
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

// SetRegistry replaces the tool registry at runtime.
func (a *Agent) SetRegistry(registry *tools.Registry) {
	a.registry = registry
}

// SetClientConfig stores model and pricing configuration for session recreation.
func (a *Agent) SetClientConfig(models []string, enablePricing bool) {
	a.models = models
	a.enablePricing = enablePricing
}

// RecreateSession creates a new OmniGo session with the given system prompt
// and tool list, preserving existing conversation history.
// This is used when activating a skill to swap the LLM configuration.
func (a *Agent) RecreateSession(systemPrompt string, toolNames []string, models []string) error {
	if len(models) == 0 {
		models = a.models
	}
	if len(models) == 0 {
		return fmt.Errorf("at least one model is required")
	}

	client, err := NewOmniGoClientWithTools(models, a.enablePricing, systemPrompt, toolNames)
	if err != nil {
		return fmt.Errorf("recreating session: %w", err)
	}

	a.client = client
	return nil
}

// SetSend sets the callback function for sending messages to the TUI.
// This is called after the tea.Program is created to wire p.Send.
func (a *Agent) SetSend(fn SendFunc) {
	a.send = fn
}

// ModelName returns the active model name from the LLM client.
func (a *Agent) ModelName() string {
	if a.client == nil {
		return "no-api-key"
	}
	return a.client.ModelName()
}

// Session returns the agent's history session.
func (a *Agent) Session() *history.Session {
	return a.session
}
