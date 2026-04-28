package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/Zheng5005/omnigo"
)

// Tool schema structs for OmniGo registration.
// Names are chosen so OmniGo's structName() (lowercase-first) produces names
// matching our tool registry: list_files, grep_search, read_file, run_command.

// List_files is the OmniGo tool schema for the list_files tool.
type List_files struct {
	Path     string `json:"path" desc:"Root path to list from. Defaults to current directory."`
	MaxDepth int    `json:"max_depth" desc:"Maximum depth to traverse. -1 for unlimited. Default 3."`
}

// Grep_search is the OmniGo tool schema for the grep_search tool.
type Grep_search struct {
	Pattern string `json:"pattern" desc:"Regex pattern to search for."`
	Path    string `json:"path" desc:"Root path to search from. Defaults to current directory."`
	Include string `json:"include" desc:"File glob filter, e.g. *.go"`
}

// Read_file is the OmniGo tool schema for the read_file tool.
type Read_file struct {
	Path      string `json:"path" desc:"Path to the file to read."`
	StartLine int    `json:"start_line" desc:"Start line number (1-indexed, inclusive)."`
	EndLine   int    `json:"end_line" desc:"End line number (1-indexed, inclusive)."`
}

// Run_command is the OmniGo tool schema for the run_command tool.
type Run_command struct {
	Command string `json:"command" desc:"Shell command to execute."`
}

// OmniGoClient adapts the OmniGo library to the LLMClient interface.
// It uses OmniGo's synchronous Chat() for all calls since streaming does not
// support tool calls. Text responses are delivered via onChunk as simulated
// streaming.
type OmniGoClient struct {
	client    *omnigo.Client
	session   *omnigo.Session
	models    []string
	enablePricing bool
	systemPrompt string
	tools     []interface{}
	modelName string
}

// NewOmniGoClient creates a new OmniGoClient with the given model priority.
// The first model in the list is used as the primary; the rest are fallbacks.
// API keys are resolved from environment variables by OmniGo.
func NewOmniGoClient(models []string, enablePricing bool, systemPrompt string) (*OmniGoClient, error) {
	return NewOmniGoClientWithTools(models, enablePricing, systemPrompt, nil)
}

// NewOmniGoClientWithTools creates a client with only the specified tools.
// toolNames is a list of tool names to register (e.g., ["list_files", "read_file"]).
// If toolNames is nil or empty, registers all default tools.
func NewOmniGoClientWithTools(models []string, enablePricing bool, systemPrompt string, toolNames []string) (*OmniGoClient, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("at least one model is required")
	}

	omniModels := make([]omnigo.Model, len(models))
	for i, m := range models {
		omniModels[i] = omnigo.Model(m)
	}

	client, err := omnigo.NewClient(omnigo.Config{
		Models:               omniModels,
		EnableDynamicPricing: enablePricing,
	})
	if err != nil {
		return nil, fmt.Errorf("creating OmniGo client: %w", err)
	}

	session := client.NewSession(omnigo.WithSystemPrompt(systemPrompt))

	if len(toolNames) == 0 {
		toolNames = []string{"list_files", "grep_search", "read_file", "run_command"}
	}

	toolMap := map[string]interface{}{
		"list_files":  List_files{},
		"grep_search": Grep_search{},
		"read_file":   Read_file{},
		"run_command": Run_command{},
	}

	toolsToRegister := make([]interface{}, 0, len(toolNames))
	for _, name := range toolNames {
		if tool, ok := toolMap[name]; ok {
			toolsToRegister = append(toolsToRegister, tool)
		}
	}

	if len(toolsToRegister) > 0 {
		if err := session.RegisterTools(toolsToRegister...); err != nil {
			return nil, fmt.Errorf("registering tools: %w", err)
		}
	}

	return &OmniGoClient{
		client:       client,
		session:      session,
		models:       models,
		enablePricing: enablePricing,
		systemPrompt: systemPrompt,
		tools:        toolsToRegister,
		modelName:    models[0],
	}, nil
}

// ModelName returns the name of the primary model.
func (o *OmniGoClient) ModelName() string {
	return o.modelName
}

// ChatStream implements LLMClient. It uses OmniGo's synchronous Chat() to
// support tool calls (OmniGo's streaming API does not return tool calls).
// Text is delivered through onChunk to simulate streaming for the TUI.
//
// For tool results, a fresh session is created to avoid accumulated history
// issues with strict providers like Gemini. The full conversation context
// is passed as a single formatted message.
func (o *OmniGoClient) ChatStream(ctx context.Context, messages []ChatMessage, toolDefs []ToolDefinition, onChunk func(StreamChunk)) (*ChatMessage, *Usage, error) {
	lastMsg := messages[len(messages)-1]
	var input string
	useFreshSession := false

	if lastMsg.Role == "tool" {
		// For tool results, use a fresh session to avoid Gemini format errors.
		// Include the original user prompt as context in a single message.
		originalPrompt := findLastUserMessage(messages)
		toolResults := formatToolResults(messages)
		if originalPrompt != "" {
			input = fmt.Sprintf("Original request: %s\n\n%s", originalPrompt, toolResults)
		} else {
			input = toolResults
		}
		useFreshSession = true
	} else {
		input = lastMsg.Content
	}

	// Ensure input is never empty — Gemini rejects empty messages
	if strings.TrimSpace(input) == "" {
		input = "(no input provided)"
	}

	// Use a fresh session for tool results to avoid accumulated history issues
	var sess *omnigo.Session
	if useFreshSession && o.client != nil {
		sess = o.client.NewSession(omnigo.WithSystemPrompt(o.systemPrompt))
		if len(o.tools) > 0 {
			_ = sess.RegisterTools(o.tools...)
		}
	} else {
		sess = o.session
	}

	resp, err := sess.Chat(ctx, input)
	if err != nil {
		return nil, nil, err
	}

	// Deliver text to TUI via onChunk (simulated streaming).
	if resp.Text != "" {
		onChunk(StreamChunk{Content: resp.Text})
	}

	result := &ChatMessage{
		Role:    "assistant",
		Content: resp.Text,
	}

	for _, tc := range resp.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Name,
			Arguments: string(tc.Arguments),
		})
	}

	usage := &Usage{
		PromptTokens:     resp.Usage.InputTokens,
		CompletionTokens: resp.Usage.OutputTokens,
		TotalTokens:      resp.Usage.TotalTokens,
		Cost:             resp.Cost.TotalCost,
		InputCost:        resp.Cost.InputCost,
		OutputCost:       resp.Cost.OutputCost,
	}

	return result, usage, nil
}

// findLastUserMessage returns the content of the most recent user-role message.
func findLastUserMessage(messages []ChatMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i].Content
		}
	}
	return ""
}

// formatToolResults builds a user message containing tool execution results
// from the trailing tool-role messages in the conversation.
// Format is structured text that all providers (including Gemini) can parse.
func formatToolResults(messages []ChatMessage) string {
	var b strings.Builder
	b.WriteString("The following tool calls were executed. Use these results to continue:\n\n")

	count := 0
	for _, m := range messages {
		if m.Role != "tool" {
			continue
		}
		count++
		content := m.Content
		if content == "" {
			content = "(empty result)"
		}
		// Sanitize: truncate very long results to avoid token limits
		if len(content) > 8000 {
			content = content[:8000] + "\n... (truncated)"
		}
		b.WriteString(fmt.Sprintf("## Tool Result %d\n%s\n\n", count, content))
	}

	if count == 0 {
		b.WriteString("(no tool results available)\n")
	}

	return b.String()
}

// DefaultSystemPrompt is the default system prompt used when none is provided.
const DefaultSystemPrompt = `You are OmniCLI, an AI coding assistant running in the terminal. You help developers understand and work with their codebase.

You have access to the following tools:
- list_files: List project files in a tree format
- grep_search: Search file contents using regex patterns
- read_file: Read file content with optional line range
- run_command: Execute shell commands (requires user approval)

Use tools to explore the codebase before answering questions. Be concise and helpful.
When using tools, provide the arguments as a JSON object matching the tool's parameter schema.`
