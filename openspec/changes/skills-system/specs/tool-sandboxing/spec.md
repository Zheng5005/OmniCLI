# Tool Sandboxing Specification

## Purpose

Per-skill restriction of available tools and command safe-list patterns. Affects both the tool registry (execution) and OmniGo session (LLM schema).

## Requirements

### Requirement: Registry Tool Filtering

When a skill is activated, the tool registry MUST filter its available tools to only those listed in the skill's `tools` array.

#### Scenario: Filter to skill tools
- GIVEN the registry has 10 tools registered
- AND a skill declares `tools: ["grep_search", "read_file", "write_file"]`
- WHEN the skill is activated
- THEN `registry.List()` MUST return only the 3 declared tools
- AND `registry.Get("list_files")` MUST return an error (tool not available)

#### Scenario: Unknown tool in skill declaration
- GIVEN a skill declares `tools: ["grep_search", "nonexistent_tool"]`
- WHEN the skill is activated
- THEN `grep_search` MUST be available
- AND `nonexistent_tool` MUST be silently ignored
- AND a warning SHOULD be logged about the unknown tool

#### Scenario: No tools declared
- GIVEN a skill declares `tools: []` (empty array)
- WHEN the skill is activated
- THEN no tools MUST be available to the agent
- AND the agent MUST operate without tool access

#### Scenario: Deactivation restores full registry
- GIVEN a skill is active with 3 tools
- WHEN the skill is deactivated
- THEN `registry.List()` MUST return all 10 originally registered tools

### Requirement: OmniGo Session Tool Schema

The OmniGo session MUST be updated to reflect only the skill-scoped tools, so the LLM does not see schemas for unavailable tools.

#### Scenario: Session re-created with subset of tools
- GIVEN the original session has 10 tools registered
- AND a skill activates with 3 tools
- WHEN the session is swapped
- THEN the new session MUST have only the 3 skill tools registered
- AND the LLM MUST NOT receive schemas for the other 7 tools

#### Scenario: Conversation history preserved on swap
- GIVEN the session has 5 messages in history
- WHEN the session is swapped for tool sandboxing
- THEN all 5 messages MUST be copied to the new session
- AND no conversation context MUST be lost

#### Scenario: Deactivation restores full session tools
- GIVEN a skill is active with 3 tools
- WHEN the skill is deactivated
- THEN the session MUST be re-created with all 10 tools
- AND conversation history MUST be preserved

### Requirement: Safe List Replacement

When a skill is activated, the RunCommandTool's safe patterns MUST be replaced with the skill's `safe_list`.

#### Scenario: Skill safe list replaces defaults
- GIVEN the default safe list includes `go test`, `git status`
- AND a skill declares `safe_list: ["^ls\\s", "^cat\\s"]`
- WHEN the skill is activated
- THEN `ls foo` MUST be classified as safe
- AND `go test ./...` MUST be classified as risky (not in skill's safe list)

#### Scenario: Empty safe list
- GIVEN a skill declares `safe_list: []`
- WHEN the skill is activated
- THEN NO commands MUST be classified as safe
- AND all commands MUST require explicit `[y/n]` confirmation

#### Scenario: Invalid regex in safe list
- GIVEN a skill declares `safe_list: ["[invalid regex"]`
- WHEN the skill is loaded
- THEN the invalid pattern MUST be rejected with a warning
- AND the skill MUST still activate with the remaining valid patterns

### Requirement: Auto-Execute Safe Commands

When `auto_execute_safe` is true, safe commands MUST skip the approval prompt entirely.

#### Scenario: Auto-execute safe command
- GIVEN a skill has `auto_execute_safe: true` and `safe_list: ["^ls\\s"]`
- WHEN the agent proposes `ls -la`
- THEN the command MUST execute immediately without user confirmation
- AND the output MUST appear in the viewport

#### Scenario: Risky command still requires approval
- GIVEN a skill has `auto_execute_safe: true`
- WHEN the agent proposes `rm -rf build/`
- THEN the `[y/n]` confirmation MUST still be shown (command is not safe)

#### Scenario: Auto-execute disabled
- GIVEN a skill has `auto_execute_safe: false` (default)
- WHEN the agent proposes a safe command `ls -la`
- THEN the `[Enter to run]` prompt MUST still appear
- AND the user MUST confirm before execution

#### Scenario: Deactivation restores default safe list
- GIVEN a skill's safe list is active
- WHEN the skill is deactivated
- THEN the default safe list (go test, git status, etc.) MUST be restored
- AND `auto_execute_safe` MUST revert to false
