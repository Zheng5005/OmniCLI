# MCP Approval Specification

## Purpose

Trust-based approval model for MCP tool execution — visual confirmation for untrusted servers, silent execution with status bar notification for trusted servers.

## Requirements

### Requirement: Trust Level Enforcement

The system MUST check the `trusted` flag for each MCP server before executing its tools. Untrusted server tool calls MUST require explicit user approval. Trusted server tool calls MUST execute without interruption.

#### Scenario: Untrusted tool call requires approval

- GIVEN an MCP server is configured with `trusted: false`
- WHEN the agent proposes calling one of its tools
- THEN the TUI MUST display a visual confirmation box showing tool name, server name, and JSON arguments
- AND execution MUST pause until the user approves or denies

#### Scenario: Trusted tool call executes silently

- GIVEN an MCP server is configured with `trusted: true`
- WHEN the agent proposes calling one of its tools
- THEN the tool MUST execute immediately without user interaction
- AND the status bar MUST show a brief notification (e.g., `[✓] postgres:query`)

#### Scenario: User denies untrusted tool call

- GIVEN an untrusted tool call is awaiting approval
- WHEN the user selects "Deny"
- THEN the tool MUST NOT execute
- AND the agent MUST receive an error indicating the call was denied by the user

#### Scenario: User approves untrusted tool call

- GIVEN an untrusted tool call is awaiting approval
- WHEN the user selects "Approve"
- THEN the tool MUST execute and return results to the agent

### Requirement: Approval UI

The system MUST render an approval dialog in the TUI that is distinct from the existing command approval flow. The dialog MUST show: server name, tool name, tool description, and formatted JSON arguments.

#### Scenario: Approval dialog display

- GIVEN an untrusted MCP tool call is pending
- WHEN the approval dialog renders
- THEN it MUST show server name, tool name, description, and arguments in a readable format
- AND it MUST offer "Approve" and "Deny" options

#### Scenario: Approval dialog keyboard navigation

- GIVEN the approval dialog is visible
- WHEN the user navigates with arrow keys and presses Enter
- THEN the selected option MUST be executed

### Requirement: Trust Configuration

The system MUST allow changing a server's trust level via configuration edit or CLI command. Trust changes MUST take effect immediately for subsequent tool calls.

#### Scenario: Change trust via config edit

- GIVEN a server is currently untrusted
- WHEN the user sets `trusted: true` in `omnisettings.json`
- THEN subsequent tool calls from that server MUST execute without approval

#### Scenario: Change trust via CLI

- GIVEN a server is currently trusted
- WHEN the user runs `omni mcp trust <name> false`
- THEN subsequent tool calls from that server MUST require approval

### Requirement: Session-Level Trust Override

The system MAY support a session-level trust override that persists only for the current REPL session. This MUST NOT modify the persistent configuration file.

#### Scenario: Session trust override

- GIVEN an untrusted server in the current session
- WHEN the user chooses "Trust for this session" in the approval dialog
- THEN subsequent tool calls from that server MUST execute without approval
- AND the `trusted` flag in `omnisettings.json` MUST NOT be changed
