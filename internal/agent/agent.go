// Package agent orchestrates the LLM ↔ tool interaction loop for OmniCLI.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omnicli/omnicli/internal/history"
	"github.com/omnicli/omnicli/internal/mcp"
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
	InputCost        float64
	OutputCost       float64
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
	client           LLMClient
	registry         *tools.Registry
	session          *history.Session
	send             SendFunc
	models           []string
	enablePricing    bool
	mcpmgr           *mcp.Manager
	baseSystemPrompt string
	pinnedResources  []mcp.PinnedResource
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
// Pinned resources are automatically appended to the system prompt.
func (a *Agent) RecreateSession(systemPrompt string, toolNames []string, models []string) error {
	if systemPrompt != "" {
		a.baseSystemPrompt = systemPrompt
	}
	if len(models) == 0 {
		models = a.models
	}
	if len(models) == 0 {
		return fmt.Errorf("at least one model is required")
	}

	fullPrompt := a.buildSystemPrompt()
	client, err := NewOmniGoClientWithTools(models, a.enablePricing, fullPrompt, toolNames)
	if err != nil {
		return fmt.Errorf("recreating session: %w", err)
	}

	a.client = client
	return nil
}

// buildSystemPrompt constructs the full system prompt including pinned resources.
func (a *Agent) buildSystemPrompt() string {
	if len(a.pinnedResources) == 0 {
		return a.baseSystemPrompt
	}

	var b strings.Builder
	b.WriteString(a.baseSystemPrompt)
	b.WriteString("\n\n---\n\n## Attached Resources\n\n")
	for _, r := range a.pinnedResources {
		b.WriteString(fmt.Sprintf("### %s (from %s)\n", r.Name, r.ServerName))
		if r.Description != "" {
			b.WriteString(fmt.Sprintf("Description: %s\n", r.Description))
		}
		b.WriteString(fmt.Sprintf("URI: %s\n\n", r.URI))
		b.WriteString(r.Content)
		b.WriteString("\n\n---\n\n")
	}
	return b.String()
}

// AttachResource adds a pinned resource and updates the system prompt.
func (a *Agent) AttachResource(res mcp.PinnedResource) error {
	for _, r := range a.pinnedResources {
		if r.ServerName == res.ServerName && r.URI == res.URI {
			return fmt.Errorf("resource already pinned")
		}
	}
	a.pinnedResources = append(a.pinnedResources, res)
	return nil
}

// DetachResource removes a pinned resource by server name and URI.
func (a *Agent) DetachResource(serverName, uri string) {
	filtered := make([]mcp.PinnedResource, 0, len(a.pinnedResources))
	for _, r := range a.pinnedResources {
		if r.ServerName != serverName || r.URI != uri {
			filtered = append(filtered, r)
		}
	}
	a.pinnedResources = filtered
}

// PinnedResources returns a copy of the currently pinned resources.
func (a *Agent) PinnedResources() []mcp.PinnedResource {
	result := make([]mcp.PinnedResource, len(a.pinnedResources))
	copy(result, a.pinnedResources)
	return result
}

// SetBaseSystemPrompt sets the base system prompt (without resource injection).
func (a *Agent) SetBaseSystemPrompt(prompt string) {
	a.baseSystemPrompt = prompt
}

// SetSend sets the callback function for sending messages to the TUI.
// This is called after the tea.Program is created to wire p.Send.
func (a *Agent) SetSend(fn SendFunc) {
	a.send = fn
}

// SetMCPManager sets the MCP manager for dynamic tool registration.
func (a *Agent) SetMCPManager(mgr *mcp.Manager) {
	a.mcpmgr = mgr
}

// MCPManager returns the agent's MCP manager, or nil if not set.
func (a *Agent) MCPManager() *mcp.Manager {
	return a.mcpmgr
}

// AllToolNames returns the names of all tools currently registered.
func (a *Agent) AllToolNames() []string {
	registered := a.registry.List()
	names := make([]string, len(registered))
	for i, t := range registered {
		names[i] = t.Name()
	}
	return names
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
