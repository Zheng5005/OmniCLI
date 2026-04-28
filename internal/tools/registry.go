package tools

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

// Tool defines the interface for all agent tools.
type Tool interface {
	// Name returns the unique name of the tool.
	Name() string
	// Description returns a human-readable description of what the tool does.
	Description() string
	// Parameters returns the JSON Schema describing the tool's accepted arguments.
	Parameters() json.RawMessage
	// Execute runs the tool with the given arguments and returns the result.
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

// Registry holds registered tools and provides lookup by name.
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry. If a tool with the same name
// already exists, it is replaced.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get retrieves a tool by name. Returns the tool and true if found,
// or nil and false if not found.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools sorted by name.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

// Filter returns a new Registry containing only tools whose names
// appear in allowedTools. If allowedTools is nil or empty, returns
// a copy of the full registry.
func (r *Registry) Filter(allowedTools []string) *Registry {
	r.mu.RLock()
	defer r.mu.RUnlock()

	filtered := NewRegistry()

	if len(allowedTools) == 0 {
		for name, t := range r.tools {
			filtered.tools[name] = t
		}
		return filtered
	}

	for _, name := range allowedTools {
		if t, ok := r.tools[name]; ok {
			filtered.tools[name] = t
		}
	}
	return filtered
}

// Deregister removes all tools whose names start with the given prefix
// followed by "__". This is used to remove tools from a specific MCP
// server when it disconnects.
func (r *Registry) Deregister(prefix string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pattern := prefix + "__"
	for name := range r.tools {
		if strings.HasPrefix(name, pattern) {
			delete(r.tools, name)
		}
	}
}
