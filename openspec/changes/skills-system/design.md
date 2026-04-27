# Design: Skills System

## Technical Approach

Introduce `internal/skills/` as a self-contained plugin package that loads JSON skill definitions, resolves template variables via wizard overlay, and reconfigures the agent at runtime. The existing TUI state machine gains two new states (`StateSkillWizard`, `StateSkillListing`). Slash commands are generalized from the hardcoded `/exit` check into a `SlashRouter` dispatch map. Tool sandboxing is implemented via a filtered `Registry` view (execution side) and a new OmniGo session with subset tools (LLM schema side). The skill generator runs as a separate Bubble Tea program under `omni skill init`.

## Architecture Decisions

### Decision: Session Hot-Swap Strategy

**Choice**: Recreate OmniGo session with subset tools, copying conversation history
**Alternatives**: (A) Keep session, modify tool registrations dynamically; (B) Per-skill sessions from scratch
**Rationale**: OmniGo's `Session` does not expose `UnregisterTools()`. Option A is impossible without upstream changes. Option B loses context. Copying messages to a new session gives a clean tool scope while preserving conversation continuity.

### Decision: Allowed Tool Filter Location

**Choice**: Add `Filter(names []string) *Registry` on `tools.Registry` producing a read-only subset view; `Agent.allowedTools` field gates `buildToolDefs()` and `executeTool()`
**Alternatives**: (A) Remove tools from the original registry; (B) Build allow-list in Agent only
**Rationale**: Option A mutates shared state — the original registry is needed when the skill deactivates. Option B couples filtering logic into Agent alone. A `Filter` method produces an immutable snapshot that Agent can hold and swap; the original registry remains intact for deactivation.

### Decision: Variable Wizard as Bubble Tea Sub-Model

**Choice**: Dedicated `SkillWizardModel` with its own Update/View, managed by root Model as an overlay state
**Alternatives**: (A) Prompt variables inline in the textarea; (B) External CLI flag pre-fill
**Rationale**: Inline prompts would require parsing partial user input and conflict with the existing submit flow. Flag pre-fill can't handle dynamic variables discovered at skill-load time. A sub-model keeps the wizard self-contained with clear enter/exit transitions, following Bubble Tea's Elm Architecture conventions already used in the codebase.

### Decision: Skill File Resolution Order

**Choice**: Project-local `.omnicli/skills/` first, then global `~/.config/omnicli/skills/`; first match wins
**Alternatives**: Merge both directories with global as fallback for missing fields
**Rationale**: First-match is simpler, predictable, and mirrors how `omnisettings.json` resolution works (project overrides global). Field-level merging adds complexity for minimal benefit — users can copy and customize.

### Decision: Slash Router as Map

**Choice**: `SlashRouter` struct with `map[string]SlashHandler` dispatch, checked in `Model.Update` before normal input handling
**Alternatives**: (A) Interface-based command pattern; (B) Switch statement
**Rationale**: The map approach is a minimal generalization of the existing `/exit` switch case. Adding new commands is a single `router.Register()` call. An interface pattern is overkill for the current command count (~3). A switch statement would just move the hardcoding.

## Data Flow

Skill activation flow:

```
User types "/skill docs-expert"
         │
    ┌────▼────┐
    │ SlashRouter │──→ dispatches "skill" command with args "docs-expert"
    └────┬────┘
         │
    ┌────▼────┐
    │ SkillRegistry │──→ Load("docs-expert") from .omnicli/skills/ or ~/.config/omnicli/skills/
    └────┬────┘
         │
    ┌────▼─────────┐
    │ Has {{vars}}? │── Yes ──→ StateSkillWizard (SkillWizardModel overlay)
    └────┬─────────┘              │
         │ No                    │ User fills variables
         │                       ▼
    ┌────▼──────────────┐   SkillActivatedMsg
    │ SkillActivatedMsg   │
    │ {name, prompt,     │
    │  tools, safePatterns│
    │  autoExecuteSafe}  │
    └────┬──────────────┘
         │
    ┌────▼──────────────────────────────────┐
    │ Model.Update handles SkillActivatedMsg │
    │  1. agent.SetAllowedTools(skill.Tools) │
    │  2. agent.SetSystemPrompt(resolved)    │──→ recreates OmniGo session
    │  3. RunCommandTool.SetSafePatterns(...)│
    │  4. statusBar.SetSkill(skill.DisplayName)│
    │  5. input.SetPromptPrefix(prefix)      │
    └───────────────────────────────────────┘
         │
    ┌────▼────┐
    │ StateNormal │──→ agent now uses filtered tools + custom prompt
    └─────────┘
```

Deactivation flow:

```
User types "/skill off"  (or "/exit" which deactivates first)
         │
    SlashRouter ──→ SkillDeactivateMsg
         │
    Model.Update:
      1. agent.SetAllowedTools(nil)  ──→ restores full tool set
      2. agent.SetSystemPrompt(default) ──→ recreates OmniGo session with all tools
      3. RunCommandTool.SetSafePatterns(originalPatterns)
      4. statusBar.SetSkill("")
      5. input.SetPromptPrefix("")
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/skills/skill.go` | Create | `Skill` struct, JSON schema, `Load()` from file, `Validate()` |
| `internal/skills/resolver.go` | Create | Dual-path resolution: project dir → global dir |
| `internal/skills/template.go` | Create | `ExtractVariables()` regex, `Resolve()` using `text/template` |
| `internal/skills/registry.go` | Create | `SkillRegistry` cache: `Register()`, `LoadAll()`, `Get(name)` |
| `internal/skills/skill_test.go` | Create | Tests for load, validate, template resolution |
| `internal/skills/resolver_test.go` | Create | Tests for dual-path resolution |
| `internal/skills/template_test.go` | Create | Tests for variable extraction and injection |
| `internal/skills/generator.go` | Create | Bubble Tea model for `omni skill init` interactive wizard |
| `internal/tui/slashrouter.go` | Create | `SlashRouter` type and `SlashHandler` func type |
| `internal/tui/skillwizard.go` | Create | `SkillWizardModel` Bubble Tea sub-model for variable input |
| `internal/tui/model.go` | Modify | Add `StateSkillWizard`, `StateSkillListing`; route through `SlashRouter`; handle skill activation/deactivation messages; add `skillRegistry` and `originalSafePatterns` fields |
| `internal/tui/input.go` | Modify | Add `promptPrefix` field and `SetPromptPrefix()`; render prefix in `View()` |
| `internal/tui/statusbar.go` | Modify | Add `skillName` field and `SetSkill(name)`; render `SKILL: <name>` segment |
| `internal/tui/messages.go` | Modify | Add `SkillActivateMsg`, `SkillActivatedMsg`, `SkillDeactivateMsg`, `SkillListMsg` |
| `internal/agent/agent.go` | Modify | Add `allowedTools []string`, `systemPrompt string` fields; `SetAllowedTools()`, `SetSystemPrompt()`, `SetSafePatterns()` methods; gate `buildToolDefs()` and `executeTool()` |
| `internal/agent/omnigo.go` | Modify | Add `NewSessionWithTools()` that creates a new OmniGo session with a subset of tool schemas, preserving message history |
| `internal/tools/registry.go` | Modify | Add `Filter(names []string) *Registry` method returning a subset view |
| `internal/tools/run_command.go` | Modify | Add `SetSafePatterns([]*regexp.Regexp)` and `SetAutoExecuteSafe(bool)` methods |
| `cmd/omni/main.go` | Modify | Add subcommand parsing (`skill init`), wire `SkillRegistry`, register slash commands, pass registry to model |

## Interfaces / Contracts

### Skill struct (`internal/skills/skill.go`)

```go
type Skill struct {
    Name            string            `json:"name"`
    DisplayName     string            `json:"display_name"`
    Description     string            `json:"description"`
    SystemPrompt    string            `json:"system_prompt"`
    Tools           []string          `json:"tools"`
    SafeList        []string          `json:"safe_list"`
    AutoExecuteSafe bool              `json:"auto_execute_safe"`
    Variables       map[string]string `json:"variables"` // name → description
}
```

JSON example:
```json
{
  "name": "docs-expert",
  "display_name": "Docs Expert",
  "description": "Specialized in documentation writing",
  "system_prompt": "You are a documentation expert for {{project_type}}. Focus on {{focus_area}}.",
  "tools": ["read_file", "grep_search", "list_files"],
  "safe_list": ["^cat\\s", "^head\\s"],
  "auto_execute_safe": true,
  "variables": {
    "project_type": "Type of project (e.g., Go library, React app)",
    "focus_area": "Documentation focus (e.g., API docs, tutorials)"
  }
}
```

### SkillRegistry (`internal/skills/registry.go`)

```go
type SkillRegistry struct {
    skills map[string]*Skill
}

func NewSkillRegistry() *SkillRegistry
func (r *SkillRegistry) Register(skill *Skill)
func (r *SkillRegistry) Get(name string) (*Skill, bool)
func (r *SkillRegistry) LoadAll(resolver *Resolver) error
func (r *SkillRegistry) List() []*Skill
```

### Resolver (`internal/skills/resolver.go`)

```go
type Resolver struct {
    projectDir string // .omnicli/skills/
    globalDir  string // ~/.config/omnicli/skills/
}

func NewResolver() (*Resolver, error)
func (r *Resolver) Resolve(name string) (string, error) // returns file path, project-first
func (r *Resolver) ListAvailable() ([]string, error)     // list all .json skill files
```

### Template Engine (`internal/skills/template.go`)

```go
func ExtractVariables(prompt string) []string           // regex: {{\w+}}
func Resolve(prompt string, vars map[string]string) (string, error)
func ValidateVariableNames(skill *Skill) []error        // cross-check variables vs template
```

### SlashRouter (`internal/tui/slashrouter.go`)

```go
type SlashHandler func(args string) tea.Msg

type SlashRouter struct {
    handlers map[string]SlashHandler
}

func NewSlashRouter() *SlashRouter
func (r *SlashRouter) Register(cmd string, handler SlashHandler)
func (r *SlashRouter) Dispatch(input string) (tea.Msg, bool) // returns (msg, handled)
```

### Tool Registry Filter (`internal/tools/registry.go` addition)

```go
// Filter returns a new Registry containing only the named tools.
// Unknown names are silently skipped. The original registry is untouched.
func (r *Registry) Filter(names []string) *Registry
```

### Agent Extensions (`internal/agent/agent.go` additions)

```go
// Fields added to Agent struct:
type Agent struct {
    // ... existing fields ...
    allowedTools    []string          // nil means all tools allowed
    systemPrompt    string            // current system prompt
}

func (a *Agent) SetAllowedTools(names []string)
func (a *Agent) SetSystemPrompt(prompt string) error  // recreates OmniGo session
func (a *Agent) AllowedTools() []string
```

### OmniGo Session Swap (`internal/agent/omnigo.go` additions)

```go
// recreateSession builds a new OmniGo session with the given system prompt
// and only the specified tool schemas. Copies existing message history.
func (o *OmniGoClient) recreateSession(systemPrompt string, toolNames []string) error

// availableToolSchemas maps tool names to their OmniGo struct instances.
var availableToolSchemas = map[string]any{
    "list_files":   List_files{},
    "grep_search":  Grep_search{},
    "read_file":    Read_file{},
    "run_command":  Run_command{},
}
```

### RunCommandTool Extensions (`internal/tools/run_command.go` additions)

```go
func (t *RunCommandTool) SetSafePatterns(patterns []*regexp.Regexp)
func (t *RunCommandTool) SetAutoExecuteSafe(auto bool)
```

When `autoExecuteSafe` is true and `Classification == Safe`, the approval callback is bypassed entirely (auto-approve).

### SkillWizardModel (`internal/tui/skillwizard.go`)

```go
type SkillWizardModel struct {
    skill      *skills.Skill
    step       int                              // current variable index
    inputs     []textinput.Model                // one per variable
    variables  map[string]string                // collected values
    submitted  bool
    cancelled  bool
}

func NewSkillWizard(skill *skills.Skill) SkillWizardModel
func (m SkillWizardModel) Init() tea.Cmd
func (m SkillWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd)
func (m SkillWizardModel) View() string
```

### New TUI Messages (`internal/tui/messages.go` additions)

```go
type SkillActivateMsg struct {
    Name string // skill name from slash command
}

type SkillActivatedMsg struct {
    Name            string
    ResolvedPrompt  string
    Tools           []string
    SafePatterns    []*regexp.Regexp
    AutoExecuteSafe bool
}

type SkillDeactivateMsg struct{}

type SkillListMsg struct{}
```

### TUI State Machine Changes (`internal/tui/model.go`)

```go
const (
    StateNormal         State = iota // existing
    StateStreaming                    // existing
    StateAwaitingApproval            // existing
    StateSkillWizard                  // NEW: variable input overlay
    StateSkillListing                 // NEW: listing available skills
)
```

New fields on `Model`:
```go
type Model struct {
    // ... existing fields ...
    skillRegistry    *skills.SkillRegistry
    slashRouter      *SlashRouter
    wizard           SkillWizardModel
    activeSkill      *skills.Skill
    originalSafePatterns []*regexp.Regexp  // saved before skill activation
}
```

`InputModel` addition:
```go
type InputModel struct {
    textarea     textarea.Model
    enabled      bool
    promptPrefix string  // NEW: e.g. "[Docs Expert] "
}
```

`StatusBarModel` addition:
```go
type StatusBarModel struct {
    // ... existing fields ...
    skillName string  // NEW: displayed when non-empty
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Unit | Skill JSON load/validate | Table-driven tests with valid/invalid JSON files in `testdata/` |
| Unit | Template variable extraction | Test `ExtractVariables()` with `{{name}}` patterns, nested, edge cases |
| Unit | Template resolution | Test `Resolve()` against system prompts with provided/missing vars |
| Unit | Dual-path resolver | Test project-first resolution with temp directories; test missing files |
| Unit | Registry Filter | Test `Filter()` with known/unknown tool names; verify original registry untouched |
| Unit | SlashRouter dispatch | Test known commands return handlers; unknown commands return `handled=false` |
| Unit | RunCommandTool pattern swap | Test `SetSafePatterns()` replaces classification behavior |
| Integration | Agent tool filtering | Agent with `SetAllowedTools(["list_files"])` — verify `buildToolDefs()` only returns that tool; verify `executeTool()` rejects others |
| Integration | OmniGo session recreation | Create session with all tools, call `recreateSession()` with subset, verify new session has only those tools and preserved messages |
| Integration | Wizard → activation flow | Simulate `SkillActivateMsg`, mock skill with variables, verify wizard collects inputs and fires `SkillActivatedMsg` |
| E2E | Full skill activation | Start TUI, type `/skill docs-expert`, fill wizard variables, verify status bar, prompt prefix, and tool scope |

## Migration / Rollout

No migration required. The skills system is purely additive:
- Skill JSON files are inert without the loading code
- Existing `/exit` behavior is preserved through the `SlashRouter`
- Agent without `allowedTools` set behaves identically to current behavior (all tools allowed)
- `RunCommandTool.SetSafePatterns(nil)` restores original classification behavior
- Feature can be rolled back by reverting the single commit introducing `internal/skills/` and TUI/agent modifications

## Open Questions

- [ ] Should skill deactivation (`/skill off`) preserve or discard conversation history from the skill session? Current design: preserve (same OmniGo session, just widens tool scope back).
- [ ] Should `omni skill init` validate tool names against the current registry at generation time? Proposed: yes, warn on unknown names but allow saving.
- [ ] How to handle a skill that references a tool not in the current binary (e.g., future plugin tools)? Proposed: skip unknown tools at activation time, log a warning.