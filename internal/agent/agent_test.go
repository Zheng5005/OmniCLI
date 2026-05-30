package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/omnicli/omnicli/internal/history"
	"github.com/omnicli/omnicli/internal/mcp"
	"github.com/omnicli/omnicli/internal/tools"
)

// mockLLMClient implements LLMClient for testing.
type mockLLMClient struct {
	responses []*ChatMessage
	usages    []*Usage
	callCount int
}

func (m *mockLLMClient) ChatStream(ctx context.Context, messages []ChatMessage, toolDefs []ToolDefinition, onChunk func(StreamChunk)) (*ChatMessage, *Usage, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	idx := m.callCount
	m.callCount++
	if idx >= len(m.responses) {
		return nil, nil, fmt.Errorf("no more responses")
	}
	resp := m.responses[idx]
	if resp.Content != "" {
		onChunk(StreamChunk{Content: resp.Content})
	}
	var usage *Usage
	if idx < len(m.usages) {
		usage = m.usages[idx]
	}
	return resp, usage, nil
}

func (m *mockLLMClient) ModelName() string { return "mock-model" }

// mockTool implements tools.Tool for testing.
type mockTool struct {
	name   string
	result string
}

func (t *mockTool) Name() string                                                    { return t.name }
func (t *mockTool) Description() string                                             { return "mock tool" }
func (t *mockTool) Parameters() json.RawMessage                                     { return json.RawMessage(`{}`) }
func (t *mockTool) Execute(_ context.Context, _ json.RawMessage) (string, error) { return t.result, nil }

// collector captures messages sent by the agent.
type collector struct {
	mu   sync.Mutex
	msgs []interface{}
}

func (c *collector) send(msg interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, msg)
}

func (c *collector) get() []interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make([]interface{}, len(c.msgs))
	copy(cp, c.msgs)
	return cp
}

func TestAgentRun(t *testing.T) {
	tests := []struct {
		name           string
		prompt         string
		responses      []*ChatMessage
		usages         []*Usage
		tools          []*mockTool
		wantCallCount  int
		wantMsgTypes   []string // simplified type names in order
		wantSessionLen int
	}{
		{
			name:   "simple text response",
			prompt: "Hello",
			responses: []*ChatMessage{
				{Role: "assistant", Content: "Hello!"},
			},
			usages:         []*Usage{{Cost: 0.01}},
			wantCallCount:  1,
			wantMsgTypes:   []string{"StreamChunkMsg", "AgentDoneMsg", "CostUpdateMsg"},
			wantSessionLen: 2, // user + assistant
		},
		{
			name:   "tool call loop",
			prompt: "find TODOs",
			responses: []*ChatMessage{
				{
					Role: "assistant",
					ToolCalls: []ToolCall{
						{ID: "1", Name: "grep_search", Arguments: `{"pattern":"TODO"}`},
					},
				},
				{Role: "assistant", Content: "Found results"},
			},
			usages: []*Usage{{Cost: 0.01}, {Cost: 0.02}},
			tools: []*mockTool{
				{name: "grep_search", result: "file.go:1: // TODO fix"},
			},
			wantCallCount:  2,
			wantMsgTypes:   []string{"ToolCallMsg", "StreamChunkMsg", "AgentDoneMsg", "CostUpdateMsg"},
			wantSessionLen: 2, // user + assistant (tool messages are in-flight only)
		},
		{
			name:   "unknown tool",
			prompt: "do something",
			responses: []*ChatMessage{
				{
					Role: "assistant",
					ToolCalls: []ToolCall{
						{ID: "1", Name: "nonexistent_tool", Arguments: `{}`},
					},
				},
				{Role: "assistant", Content: "Sorry, couldn't find that tool"},
			},
			wantCallCount:  2,
			wantMsgTypes:   []string{"ToolCallMsg", "StreamChunkMsg", "AgentDoneMsg", "CostUpdateMsg"},
			wantSessionLen: 2,
		},
		{
			name:           "LLM error",
			prompt:         "fail",
			responses:      []*ChatMessage{}, // no responses → triggers error
			wantCallCount:  1,
			wantMsgTypes:   []string{"ErrorMsg"},
			wantSessionLen: 1, // only user message, no assistant
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockLLMClient{
				responses: tt.responses,
				usages:    tt.usages,
			}

			registry := tools.NewRegistry()
			for _, mt := range tt.tools {
				registry.Register(mt)
			}

			session := history.NewSession(t.TempDir())
			c := &collector{}

			ag := New(client, registry, session, c.send)
			ag.Run(context.Background(), tt.prompt)

			if client.callCount != tt.wantCallCount {
				t.Errorf("callCount = %d, want %d", client.callCount, tt.wantCallCount)
			}

			sent := c.get()
			gotTypes := make([]string, len(sent))
			for i, msg := range sent {
				switch msg.(type) {
				case StreamChunkMsg:
					gotTypes[i] = "StreamChunkMsg"
				case ToolCallMsg:
					gotTypes[i] = "ToolCallMsg"
				case AgentDoneMsg:
					gotTypes[i] = "AgentDoneMsg"
				case CostUpdateMsg:
					gotTypes[i] = "CostUpdateMsg"
				case ErrorMsg:
					gotTypes[i] = "ErrorMsg"
				default:
					gotTypes[i] = fmt.Sprintf("unknown(%T)", msg)
				}
			}

			if len(gotTypes) != len(tt.wantMsgTypes) {
				t.Fatalf("sent %d messages %v, want %d %v", len(gotTypes), gotTypes, len(tt.wantMsgTypes), tt.wantMsgTypes)
			}
			for i := range gotTypes {
				if gotTypes[i] != tt.wantMsgTypes[i] {
					t.Errorf("msg[%d] type = %s, want %s", i, gotTypes[i], tt.wantMsgTypes[i])
				}
			}

			if len(session.Messages) != tt.wantSessionLen {
				t.Errorf("session has %d messages, want %d", len(session.Messages), tt.wantSessionLen)
			}
		})
	}
}

func TestSetRegistry(t *testing.T) {
	oldRegistry := tools.NewRegistry()
	oldRegistry.Register(&mockTool{name: "old_tool", result: "old"})

	newRegistry := tools.NewRegistry()
	newRegistry.Register(&mockTool{name: "new_tool", result: "new"})

	session := history.NewSession(t.TempDir())
	ag := New(&mockLLMClient{}, oldRegistry, session, nil)

	if ag.registry != oldRegistry {
		t.Error("expected old registry to be set")
	}

	ag.SetRegistry(newRegistry)

	if ag.registry != newRegistry {
		t.Error("expected registry to be replaced")
	}

	// Verify the new registry is used.
	tool, ok := ag.registry.Get("new_tool")
	if !ok {
		t.Fatal("expected new_tool to be found in new registry")
	}
	if tool.Name() != "new_tool" {
		t.Errorf("tool name = %s, want new_tool", tool.Name())
	}
}

func TestRecreateSession(t *testing.T) {
	t.Run("empty models without stored config returns error", func(t *testing.T) {
		session := history.NewSession(t.TempDir())
		ag := New(&mockLLMClient{}, tools.NewRegistry(), session, nil)

		err := ag.RecreateSession("test prompt", nil, nil)
		if err == nil {
			t.Fatal("expected error when no models are provided or stored")
		}
	})

	t.Run("stored models used when models param is empty", func(t *testing.T) {
		session := history.NewSession(t.TempDir())
		ag := New(&mockLLMClient{}, tools.NewRegistry(), session, nil)
		ag.SetClientConfig([]string{"gemini-1.5-flash"}, true)

		// RecreateSession may fail if no API keys are available.
		// If it succeeds, verify the client was replaced.
		err := ag.RecreateSession("test prompt", []string{"list_files"}, nil)
		if err != nil {
			// Expected when API keys are missing; verify it's a creation error.
			if ag.client == nil {
				t.Skipf("skipping: no API key available (%v)", err)
			}
			t.Fatalf("unexpected error: %v", err)
		}

		if ag.client == nil {
			t.Fatal("expected client to be set after RecreateSession")
		}
		if ag.ModelName() != "gemini-1.5-flash" {
			t.Errorf("model name = %s, want gemini-1.5-flash", ag.ModelName())
		}
	})

	t.Run("explicit models override stored config", func(t *testing.T) {
		session := history.NewSession(t.TempDir())
		ag := New(&mockLLMClient{}, tools.NewRegistry(), session, nil)
		ag.SetClientConfig([]string{"old-model"}, true)

		// Use a valid model name that OmniGo can resolve with available API keys.
		err := ag.RecreateSession("test prompt", nil, []string{"gemini-1.5-flash"})
		if err != nil {
			if ag.client == nil {
				t.Skipf("skipping: no API key available (%v)", err)
			}
			t.Fatalf("unexpected error: %v", err)
		}

		if ag.client == nil {
			t.Fatal("expected client to be set after RecreateSession")
		}
		if ag.ModelName() != "gemini-1.5-flash" {
			t.Errorf("model name = %s, want gemini-1.5-flash", ag.ModelName())
		}
	})
}

// agentMockTransport implements mcp.Transport for testing.
type agentMockTransport struct {
	result json.RawMessage
}

func (m *agentMockTransport) Send(ctx context.Context, req mcp.JSONRPCRequest) (mcp.JSONRPCResponse, error) {
	return mcp.JSONRPCResponse{Result: m.result}, nil
}

func (m *agentMockTransport) Notify(ctx context.Context, method string, params json.RawMessage) error {
	return nil
}

func (m *agentMockTransport) Close() error {
	return nil
}

func TestAgentRun_MCPToolDataCost(t *testing.T) {
	resultText := strings.Repeat("a", 400)
	raw, err := json.Marshal(mcp.CallToolResult{
		Content: []mcp.ToolContent{{Type: "text", Text: resultText}},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	mt := &agentMockTransport{result: raw}
	mcpClient := mcp.NewClient("test", mcp.ServerConfig{}, mt)
	mcpTool := mcp.NewMCPTool("test", mcp.McpToolDef{
		Name:        "echo",
		Description: "Echo",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}, mcpClient, true, nil)

	llm := &mockLLMClient{
		responses: []*ChatMessage{
			{
				Role: "assistant",
				ToolCalls: []ToolCall{
					{ID: "1", Name: "test__echo", Arguments: `{}`},
				},
			},
			{Role: "assistant", Content: "Done"},
		},
		usages: []*Usage{
			{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150, Cost: 0.01, InputCost: 0.005, OutputCost: 0.005},
			{PromptTokens: 200, CompletionTokens: 100, TotalTokens: 300, Cost: 0.02, InputCost: 0.01, OutputCost: 0.01},
		},
	}

	registry := tools.NewRegistry()
	registry.Register(mcpTool)

	session := history.NewSession(t.TempDir())
	c := &collector{}
	ag := New(llm, registry, session, c.send)
	ag.Run(context.Background(), "run")

	var lastCost *CostUpdateMsg
	for _, msg := range c.get() {
		if m, ok := msg.(CostUpdateMsg); ok {
			lastCost = &m
		}
	}
	if lastCost == nil {
		t.Fatal("expected CostUpdateMsg")
	}

	// 400 chars / 4 = 100 tokens.
	// Input rate from first usage: 0.005 / 100 = 0.00005 per token.
	// MCP data cost: 100 * 0.00005 = 0.005.
	// Total cost: 0.01 + 0.005 + 0.02 = 0.035.
	wantCost := 0.035
	if math.Abs(lastCost.Cost-wantCost) > 1e-9 {
		t.Errorf("Cost = %f, want %f", lastCost.Cost, wantCost)
	}
	if lastCost.McpDataTokens != 100 {
		t.Errorf("McpDataTokens = %d, want 100", lastCost.McpDataTokens)
	}
	wantMcpDataCost := 0.005
	if math.Abs(lastCost.McpDataCost-wantMcpDataCost) > 1e-9 {
		t.Errorf("McpDataCost = %f, want %f", lastCost.McpDataCost, wantMcpDataCost)
	}
}
