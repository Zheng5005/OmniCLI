# Proposal: Skills/Plugin System

## Intent

OmniCLI currently has a monolithic architecture: all tools are always registered, the system prompt is hardcoded, and the only slash command is `/exit`. Users need a way to give the agent specialized personas (Docs Expert, Security Auditor, etc.) with restricted toolsets, custom prompts with template variables, and per-skill command safety rules — without modifying source code.

## Scope

### In Scope
- **`internal/skills` package**: JSON loader, Skill struct, template engine, dual-path resolver (project `.omnicli/skills/` + global `~/.config/omnicli/skills/`)
- **Slash command system**: Generalize the hardcoded `/exit` check into a `SlashRouter` that dispatches `/skill`, `/exit`, and future commands
- **Tool sandboxing**: Filter the registry to only skill-listed tools; swap safe-list patterns on the RunCommandTool
- **OmniGo session hot-swap**: Re-register only skill-scoped tools on the OmniGo session
- **Variable injection wizard**: Bubble Tea sub-model that interrupts the REPL to collect `{{variable}}` values before activation
- **Visual identity**: Status bar shows `SKILL: <display_name>`, input prompt changes to `[<display_name>] >`
- **Skill generator**: `omni skill init` subcommand with interactive TUI form to build skill JSONs

### Out of Scope
- Skill marketplace / remote fetching
- Skill versioning or dependency resolution
- Theme color shifting (deferred — status bar text change is sufficient for v1)
- Skill chaining or composition
- Persisting skill state across sessions (skill deactivates on exit)

## Capabilities

### New Capabilities
- `skill-management`: Loading, resolving, and activating skill JSON files with template variable injection
- `slash-commands`: Extensible slash command routing system replacing hardcoded `/exit`
- `skill-generator`: Interactive CLI (`omni skill init`) to author skill JSON files
- `tool-sandboxing`: Per-skill tool and command safe-list restrictions

### Modified Capabilities
- `omnicli-v1`: TUI input prompt and status bar must display active skill context; REPL must support slash command interception before normal submit

## Approach

### 1. `internal/skills` Package

```
internal/skills/
├── skill.go          # Skill struct, JSON schema, Load/Resolve
├── resolver.go       # Dual-path lookup: .omnicli/skills/ + ~/.config/omnicli/skills/
├── template.go       # {{variable}} extraction and text/template injection
├── registry.go       # SkillRegistry: cache of loaded skills by name
└── skill_test.go     # Unit tests for loading, templating, resolution
```

**Skill struct** maps 1:1 to the PRD JSON schema. `Load(name)` searches project dir first, then global dir (first match wins). `ResolveVariables(prompt)` extracts all `{{...}}` tags via regex. `Inject(variables map[string]string)` uses `text/template` to produce the final system prompt.

### 2. Slash Command System

Replace the hardcoded `/exit` check in `model.go` with a `SlashRouter`:

```go
type SlashHandler func(args string) tea.Msg

type SlashRouter struct {
    handlers map[string]SlashHandler
}
```

Registered at startup:
- `/exit` → quit
- `/skill <name>` → `SkillActivateMsg{name}`
- `/skills` → list available skills (future)

The router runs in `model.go`'s `Update` before normal submit handling. Unknown slash commands render a system error in the viewport.

### 3. Tool Sandboxing with Dual Registration

The codebase has a **dual-registration pattern**: tools are registered both in the `tools.Registry` (for execution) and on the `OmniGo.Session` (for LLM schema). Sandboxing must affect both:

**Registry side**: The `Agent` gains a `SetAllowedTools([]string)` method that filters `registry.List()` output. The existing `buildToolDefs()` already iterates `registry.List()` — no change needed there. For execution, `executeTool()` checks the filtered list.

**OmniGo side**: The `OmniGoClient` gains a `SetTools(tools ...any)` method that creates a **new session** with only the specified tool schemas registered. This is called during skill activation before the first LLM call.

**Safe-list side**: The `RunCommandTool` gains a `SetSafePatterns([]*regexp.Regexp)` method. On skill activation, the skill's `safe_list` is compiled and swapped in. If `auto_execute_safe` is true, the approval flow in `model.go` auto-approves safe commands (existing `Classification == "safe"` path already does Enter-to-confirm — we just skip the prompt entirely).

### 4. Bubble Tea Variable Injection Wizard

A new sub-model `SkillWizardModel` in `internal/tui/skillwizard.go`:

- Activated when `SkillActivateMsg` arrives and the skill has unresolved variables
- Renders a focused form using `bubbles/textinput` — one field per variable with its description as placeholder
- State machine: `wizardStep` index advances on Enter; Back goes back
- On submit → fires `SkillActivatedMsg{name, resolvedPrompt, allowedTools, safePatterns, autoExecuteSafe}`
- On escape → cancels activation, returns to `StateNormal`

The main `Model` gains a `StateSkillWizard` state and a `wizard SkillWizardModel` field. While in this state, normal input is disabled and the wizard renders as an overlay above the viewport.

### 5. Visual Identity

- `StatusBarModel` gains a `skillName string` field and `SetSkill(name)` method
- When active, renders `SKILL: <name>` between model name and activity
- `InputModel` gains a `promptPrefix string` field (default `""`). When a skill is active, prefix becomes `[<display_name>] ` and renders before the textarea
- Border color shift deferred to v2

### 6. Skill Generator (`omni skill init`)

A separate subcommand in `cmd/omni/main.go` using `flag` parsing:

```
omni skill init          # interactive wizard
omni skill init --name my-skill  # pre-fill name
```

Uses a dedicated Bubble Tea model (`internal/skills/generator.go`) with steps:
1. Enter skill name and display name
2. Multi-select available tools (fetched from `tools.Registry`)
3. Enter system prompt (textarea)
4. Auto-detect `{{variables}}` from prompt, prompt for descriptions
5. Enter safe-list commands (comma-separated)
6. Toggle `auto_execute_safe`
7. Preview JSON and choose save location (project or global)

### 7. Main Wiring Changes

`cmd/omni/main.go` gains subcommand parsing:
- Default (no subcommand) → existing TUI REPL
- `skill init` → skill generator TUI
- `skill list` → print available skills (future)

The TUI model receives a `SkillRegistry` reference. On `SkillActivatedMsg`, it:
1. Calls `agent.SetAllowedTools(skill.Tools)`
2. Calls `agent.SetSafePatterns(compiledPatterns)`
3. Calls `agent.SetSystemPrompt(resolvedPrompt)` (creates new OmniGo session internally)
4. Sets `statusBar.SetSkill(skill.DisplayName)`
5. Sets `input.SetPromptPrefix("[" + skill.DisplayName + "] >")`
6. Transitions to `StateNormal`

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/skills/` | New | Entire new package: loader, resolver, template, registry, generator |
| `cmd/omni/main.go` | Modified | Subcommand parsing, skill registry wiring |
| `internal/tui/model.go` | Modified | Slash router, new states, skill activation handling |
| `internal/tui/input.go` | Modified | Prompt prefix support |
| `internal/tui/statusbar.go` | Modified | Skill name display |
| `internal/tui/skillwizard.go` | New | Variable injection sub-model |
| `internal/agent/agent.go` | Modified | `SetAllowedTools`, `SetSafePatterns`, `SetSystemPrompt` methods |
| `internal/agent/omnigo.go` | Modified | Dynamic tool re-registration on session swap |
| `internal/tools/registry.go` | Modified | `Filter([]string) *Registry` method for sandboxed view |
| `internal/tools/run_command.go` | Modified | `SetSafePatterns` and `SetAutoExecute(bool)` methods |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| OmniGo session re-registration loses conversation history | Medium | Preserve session history by copying messages to new session; or use a wrapper that swaps tool schemas without recreating the session |
| Malformed skill JSON crashes the agent | Low | Strict validation on load with descriptive errors; reject invalid skills before activation |
| Template injection security (user-crafted prompts) | Low | Skills are local JSON files authored by the user; no remote execution. Still, use `text/template` (not `html/template`) and validate variable names |
| Tool name mismatch between skill JSON and registry | Medium | Validate skill tool names against registry at load time; warn on unknown tools |
| Bubble Tea state complexity with wizard overlay | Medium | Keep wizard as a focused sub-model with clear enter/exit states; thorough state transition testing |

## Rollback Plan

The skills system is additive — no existing files are removed or modified in behavior. To rollback:
1. Revert the git commit introducing `internal/skills/` and TUI/agent changes
2. Restore original `cmd/omni/main.go` (remove subcommand parsing)
3. No data migration needed — skill JSON files in `.omnicli/skills/` and `~/.config/omnicli/skills/` are inert without the code to load them

## Dependencies

- No new external Go dependencies (uses existing `text/template`, `bubbles/textinput`, `bubbles/list`)
- OmniGo session API must support creating a new session with a subset of tools (confirmed: `client.NewSession()` + `session.RegisterTools()`)

## Success Criteria

- [ ] `omni skill init` produces a valid JSON skill file that loads correctly
- [ ] `/skill documentation-expert` activates the skill, shows variable wizard, and injects resolved prompt
- [ ] Status bar displays `SKILL: Docs Expert` and prompt shows `[Docs Expert] >`
- [ ] Agent only has access to tools listed in the skill JSON (verified by LLM not seeing other tool schemas)
- [ ] Skill's `safe_list` replaces global safe patterns for command classification
- [ ] `auto_execute_safe: true` skips approval prompt for safe commands
- [ ] All existing tests pass; new tests cover skill loading, templating, and sandboxing
