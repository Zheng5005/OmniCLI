package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"github.com/omnicli/omnicli/internal/history"
	"github.com/omnicli/omnicli/internal/tools"
)

// mockLLMClient implements LLMClient for testing.
type mockLLMClient struct {
	responses []*ChatMessage
	usages    []*Usage
	callCount int
}

func (m *mockLLMClient) ChatStream(ctx context.Context, messages []ChatMessage, toolDefs []ToolDefinition, onChunk func(StreamChunk)) (*ChatMessage, *Usage, error) {
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
