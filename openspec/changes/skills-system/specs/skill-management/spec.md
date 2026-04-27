# Skill Management Specification

## Purpose

Loading, resolving, validating, and activating skill JSON files with template variable injection. Dual-path resolution (project + global).

## Requirements

### Requirement: Skill JSON Schema

A skill file MUST conform to the following schema:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique identifier (lowercase-kebab-case, no spaces) |
| `display_name` | string | Yes | Human-readable name for UI display |
| `system_prompt` | string | Yes | Prompt template with optional `{{variable}}` placeholders |
| `variables` | object | No | Map of variable name → description string |
| `tools` | array[string] | Yes | List of tool names the skill grants access to (must match registry names) |
| `safe_list` | array[string] | No | Regex patterns for commands considered safe under this skill |
| `auto_execute_safe` | bool | No | If true, safe commands skip approval prompt (default: false) |

#### Scenario: Valid skill loads successfully
- GIVEN a JSON file at `.omnicli/skills/docs-expert.json` with all required fields
- WHEN the skill loader reads the file
- THEN the file MUST parse into a valid Skill struct
- AND no error MUST be returned

#### Scenario: Missing required field rejected
- GIVEN a JSON file missing the `name` field
- WHEN the skill loader reads the file
- THEN the loader MUST return a validation error
- AND the error MUST name the missing field

#### Scenario: Invalid tool names warned
- GIVEN a skill JSON lists a tool `"nonexistent_tool"` not in the registry
- WHEN the skill is loaded
- THEN the loader MUST return a warning for the unknown tool
- AND the skill MUST still be loadable (warnings are non-fatal)

### Requirement: Dual-Path Resolution

The system MUST resolve skills by searching two directories in order:
1. Project-local: `.omnicli/skills/*.json` (relative to current working directory)
2. Global: `~/.config/omnicli/skills/*.json`

The first match by `name` wins (project overrides global).

#### Scenario: Project skill overrides global
- GIVEN a skill `docs-expert` exists in both `.omnicli/skills/` and `~/.config/omnicli/skills/`
- WHEN the resolver looks up `docs-expert`
- THEN the project-local version MUST be returned

#### Scenario: Global-only skill found
- GIVEN a skill `security-auditor` exists only in `~/.config/omnicli/skills/`
- WHEN the resolver looks up `security-auditor`
- THEN the global version MUST be returned

#### Scenario: Skill not found
- GIVEN no skill file matches the requested name in either directory
- WHEN the resolver looks up the name
- THEN the resolver MUST return a "skill not found" error
- AND the error MUST list both searched directories

#### Scenario: No skills directory exists
- GIVEN neither `.omnicli/skills/` nor `~/.config/omnicli/skills/` exist
- WHEN the resolver lists all available skills
- THEN an empty list MUST be returned (not an error)

### Requirement: Template Variable Extraction

The system MUST extract all `{{variable}}` placeholders from the `system_prompt` field and cross-reference them with the `variables` map.

#### Scenario: Variables extracted from prompt
- GIVEN a system prompt `"You are a writer for {{project_name}} focusing on {{focus_area}}."`
- WHEN variable extraction runs
- THEN the extracted variables MUST be `["project_name", "focus_area"]`

#### Scenario: Undefined variable detected
- GIVEN a prompt contains `{{missing_var}}` not listed in the `variables` map
- WHEN the skill is validated
- THEN a warning MUST be emitted for the undefined variable

#### Scenario: No variables in prompt
- GIVEN a system prompt with no `{{...}}` placeholders
- WHEN variable extraction runs
- THEN an empty variable list MUST be returned

### Requirement: Template Injection

The system MUST replace all `{{variable}}` placeholders in the system prompt with user-provided values using Go's `text/template`.

#### Scenario: All variables resolved
- GIVEN a prompt `"Project: {{project_name}}"` and variables `{"project_name": "OmniCLI"}`
- WHEN template injection runs
- THEN the result MUST be `"Project: OmniCLI"`

#### Scenario: Unresolved variable after injection
- GIVEN a prompt `"{{name}}"` and an empty variables map
- WHEN template injection runs
- THEN the injection MUST fail with an error listing the unresolved variable

#### Scenario: Variable name sanitization
- GIVEN a variable name contains shell-injection characters (e.g., `{{;rm -rf /}}`)
- WHEN the skill JSON is loaded
- THEN the variable name MUST be rejected as invalid
- AND valid variable names MUST match `[a-zA-Z_][a-zA-Z0-9_]*`

### Requirement: Skill Registry

The system MUST maintain an in-memory cache of loaded skills, keyed by skill name.

#### Scenario: List all available skills
- GIVEN 3 skills loaded from project dir and 2 from global dir
- WHEN `List()` is called on the registry
- THEN all 5 unique skills MUST be returned with their source path

#### Scenario: Get single skill
- GIVEN a skill `docs-expert` is in the registry
- WHEN `Get("docs-expert")` is called
- THEN the skill struct MUST be returned

#### Scenario: Get non-existent skill
- GIVEN no skill named `foo` is registered
- WHEN `Get("foo")` is called
- THEN an error MUST be returned

### Requirement: Skill Activation

Activating a skill MUST produce a structured message containing the resolved prompt, allowed tools, safe patterns, and auto-execute flag.

#### Scenario: Activate skill with variables
- GIVEN a skill with 2 undefined variables
- WHEN activation is requested
- THEN a `SkillNeedsVariablesMsg` MUST be emitted with the skill and variable list
- AND the wizard MUST prompt for values before activation completes

#### Scenario: Activate skill without variables
- GIVEN a skill with no `{{...}}` placeholders
- WHEN activation is requested
- THEN a `SkillActivatedMsg` MUST be emitted immediately with resolved prompt and tool list

#### Scenario: Deactivate skill
- GIVEN a skill is currently active
- WHEN `/exit` or a deactivation command is issued
- THEN the system MUST restore the default system prompt
- AND the full tool registry MUST be re-enabled
- AND the status bar skill indicator MUST be cleared
