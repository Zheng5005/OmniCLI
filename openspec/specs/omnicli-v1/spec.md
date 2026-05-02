# OmniCLI v1 — Specifications (Source of Truth)

> Current behavior of OmniCLI. Synced from `changes/archive/2026-04-25-omnicli-v1/spec.md`, `changes/archive/2026-04-28-mcp-integration/spec.md` (2026-04-28), and `changes/archive/2026-05-02-tui-improvements/specs/omnicli-v1/spec.md` (2026-05-02).

## 0. MCP Integration Requirements

### Requirement: MCP Slash Commands

The slash router MUST support `/mcp` and `/attach` commands. `/mcp` MUST list connected MCP servers with their status, tools count, and trust level. `/attach` MUST open the resource picker for MCP resources.

#### Scenario: /mcp lists servers

- GIVEN 2 MCP servers are connected (1 trusted, 1 untrusted)
- WHEN the user types `/mcp`
- THEN the TUI MUST display each server's name, status, tool count, and trust level

#### Scenario: /attach opens resource picker

- GIVEN MCP servers with resources are connected
- WHEN the user types `/attach`
- THEN the resource picker MUST appear allowing selection of available resources

### Requirement: MCP Server Status in Status Bar

The status bar MUST display MCP server connection status alongside existing activity indicators. Each server MUST show one of: connected (green), error (red), or disconnected (gray).

#### Scenario: Server status display

- GIVEN an MCP server is connected and healthy
- WHEN the status bar renders
- THEN it MUST show the server name with a connected indicator

#### Scenario: Server error display

- GIVEN an MCP server has crashed
- WHEN the status bar renders
- THEN it MUST show the server name with an ERROR indicator

### Requirement: MCP Tool Execution in Agent Loop

The agent loop MUST detect when a tool call targets an MCP tool and delegate execution to the MCP client instead of the native tool executor. The result MUST be returned to the LLM in the same format as native tool results.

#### Scenario: Delegate MCP tool call

- GIVEN the LLM requests execution of an MCP-discovered tool
- WHEN the agent loop processes the tool call
- THEN it MUST route the call to the MCP client
- AND the result MUST be formatted and returned to the LLM

### Requirement: Economic Tracking for MCP Data

The economic tracker MUST include MCP tool response data tokens in the prompt token count. The cost formula MUST be: `TotalCost = (TokensPrompt + TokensMCP_Data) × RateInput + TokensCompletion × RateOutput`. External SSE server usage fees MUST NOT be tracked or passed through.

#### Scenario: MCP data tokens counted

- GIVEN an MCP tool returns 500 tokens of data
- WHEN the economic tracker calculates session cost
- THEN the 500 tokens MUST be added to the prompt token count
- AND cost MUST be calculated at the input rate

### Requirement: Dynamic Tool Registry

The tool registry MUST support runtime registration and deregistration of tools. When tools are added or removed, the registry MUST trigger OmniGo session recreation to update the available tool schemas sent to the LLM.

#### Scenario: Register new tool at runtime

- GIVEN an active session with 5 native tools
- WHEN an MCP server connects with 3 tools
- THEN the registry MUST contain 8 tools total
- AND the OmniGo session MUST be recreated with all 8 tool schemas

## 1. REPL & TUI Specification

### Purpose
Interactive terminal REPL built on Bubble Tea (Elm Architecture) for AI-powered developer conversations with streaming, markdown rendering, and stateful session management.

### Requirement: Bubble Tea Application Lifecycle
The application MUST implement Bubble Tea's Model interface (Init, Update, View). The program MUST initialize with a tea.Program and handle graceful shutdown on Ctrl+C or `/exit` command. On shutdown, the system MUST gracefully terminate all running MCP child processes with SIGTERM before exiting.

#### Scenario: Application startup
- GIVEN the user runs `omni` binary
- WHEN the program initializes
- THEN a Bubble Tea fullscreen app MUST launch with input area, chat viewport, and status bar

#### Scenario: Graceful shutdown with MCP servers
- GIVEN the REPL is running with 2 MCP servers connected
- WHEN the user presses Ctrl+C or types `/exit`
- THEN the application MUST save current session
- AND send SIGTERM to all MCP child processes
- AND exit cleanly after processes terminate or timeout

### Requirement: User Input Handling
The TUI MUST provide a text input area at the bottom of the screen. The input MUST support multi-line editing. Pressing Enter (without Shift) MUST submit the message.

#### Scenario: Submit user message
- GIVEN the user has typed a prompt in the input area
- WHEN the user presses Enter
- THEN the message MUST be sent to the agent loop
- AND the input area MUST clear
- AND the message MUST appear in the chat viewport as a user message

### Requirement: Streaming Response with Typewriter Effect
The system MUST render AI responses incrementally as chunks arrive via Go channels. Each chunk MUST append to the current response with a standard typewriter effect.

#### Scenario: Streaming response display
- GIVEN the agent is generating a response
- WHEN text chunks arrive from OmniGo
- THEN each chunk MUST be appended to the viewport in real-time
- AND the viewport MUST auto-scroll to show the latest content

#### Scenario: Streaming completes
- GIVEN a response is streaming
- WHEN the final chunk arrives
- THEN the complete response MUST be rendered through Glamour for full markdown formatting
- AND the input area MUST re-enable for the next prompt

### Requirement: Markdown Rendering
The system MUST render completed responses as formatted markdown using Glamour. This MUST include syntax-highlighted code blocks, tables, bold, italics, and lists.

#### Scenario: Code block rendering
- GIVEN a response contains a fenced code block with language identifier
- WHEN the response completes
- THEN the code block MUST render with syntax highlighting appropriate to the language

---

## 2. Status Bar Specification

### Purpose
Persistent footer displaying session metadata and activity state.

### Requirement: Active Model Display
The status bar MUST display the name of the currently active LLM model (e.g., "GPT-4o", "Gemini 1.5 Pro").

#### Scenario: Model display
- GIVEN a session is active with a configured model
- WHEN the status bar renders
- THEN the active model name MUST be visible in the footer

### Requirement: Live Session Cost
The status bar MUST display a running USD total for the current session, updated after each API call via OmniGo's pricing engine.

#### Scenario: Cost updates after response
- GIVEN the agent completes an API call
- WHEN OmniGo returns usage/cost data
- THEN the status bar cost MUST update to reflect the cumulative session total
- AND the format MUST be `$X.XXXX`

### Requirement: Activity Indicators
The status bar MUST show the current activity state: "Searching", "Thinking", or "Executing". During any active state (Searching, Thinking, Executing, Streaming, AwaitingApproval, Wizard, McpApproval, ResourceBrowser), an animated spinner MUST be displayed alongside the activity text label. The spinner MUST start when entering an active state and stop when returning to idle. The spinner MUST NOT display during the idle/ready state.

#### Scenario: Activity state transitions
- GIVEN the agent is processing a request
- WHEN the agent invokes a search tool
- THEN the indicator MUST show "Searching" with an animated spinner
- WHEN the agent is waiting for LLM response
- THEN the indicator MUST show "Thinking" with an animated spinner
- WHEN the agent executes a shell command
- THEN the indicator MUST show "Executing" with an animated spinner

#### Scenario: Idle state
- GIVEN no agent activity is in progress
- WHEN the user is typing or idle
- THEN the activity indicator SHOULD show "Ready" or be hidden
- AND no spinner MUST be visible

#### Scenario: Spinner stops on completion
- GIVEN the spinner is animating during a "Thinking" state
- WHEN the LLM response completes and streaming ends
- THEN the spinner MUST stop
- AND the indicator MUST return to idle

---

## 3. Project Awareness Tools Specification

### Purpose
Local search tools that provide codebase context to the AI agent without full-file ingestion.

### Requirement: list_files Tool
The system MUST implement a `list_files` tool that generates a project directory tree. It MUST respect `.omniignore` patterns (gitignore syntax). It SHOULD support a max-depth parameter.

#### Scenario: List files with omniignore
- GIVEN a project has a `.omniignore` file containing `node_modules/` and `*.log`
- WHEN `list_files` is invoked on the project root
- THEN the output MUST NOT include `node_modules/` directory or any `.log` files
- AND the output MUST list all other files in tree format

#### Scenario: List files without omniignore
- GIVEN a project has no `.omniignore` file
- WHEN `list_files` is invoked
- THEN the tool MUST list all files without filtering (except hidden dirs like `.git`)

### Requirement: grep_search Tool
The system MUST implement a `grep_search` tool supporting both keyword and regex-based search. It MUST return matching file paths, line numbers, and matching line content.

#### Scenario: Keyword search
- GIVEN a project contains files with the text "TODO"
- WHEN `grep_search` is invoked with query "TODO"
- THEN results MUST include file path, line number, and line content for each match

#### Scenario: Regex search
- GIVEN a project contains Go files
- WHEN `grep_search` is invoked with pattern `func\s+Test\w+`
- THEN results MUST return all test function declarations with file paths and line numbers

#### Scenario: No matches
- GIVEN a project with no matching content
- WHEN `grep_search` is invoked with a non-existent pattern
- THEN the tool MUST return an empty result set (not an error)

### Requirement: read_file Tool
The system MUST implement a `read_file` tool that reads file content. It MUST support optional line-range parameters (start_line, end_line) to load partial files.

#### Scenario: Read full file
- GIVEN a file exists at the specified path
- WHEN `read_file` is invoked without line range
- THEN the full file content MUST be returned

#### Scenario: Read line range
- GIVEN a file has 100 lines
- WHEN `read_file` is invoked with start_line=10, end_line=20
- THEN only lines 10-20 MUST be returned

#### Scenario: File not found
- GIVEN the specified file does not exist
- WHEN `read_file` is invoked
- THEN the tool MUST return an error message (not panic or crash)

---

## 4. Command Execution Specification

### Purpose
Shell command execution with a tiered safety model and user approval gates.

### Requirement: Safe List System
The system MUST maintain a default safe list of non-destructive command patterns. Default safe patterns MUST include:
- Go: `go test`, `go fmt`, `go build`, `go mod tidy`, `go list`
- Git: `git status`, `git diff`, `git log`, `git branch`, `git show`

Commands MUST be matched against the safe list using regex patterns compiled at config load time.

#### Scenario: Safe command identification
- GIVEN the default safe list is active
- WHEN the agent proposes `go test ./...`
- THEN the command MUST be classified as safe

#### Scenario: Risky command identification
- GIVEN the default safe list is active
- WHEN the agent proposes `rm -rf ./build`
- THEN the command MUST be classified as risky

### Requirement: Approval Gates
Safe commands MUST require a single `[Enter]` keypress to confirm. Risky or unrecognized commands MUST require explicit `[y/n]` confirmation. The user MUST be able to decline any command.

#### Scenario: Approve safe command
- GIVEN a command is classified as safe
- WHEN the TUI presents it with `[Enter to run]`
- THEN pressing Enter MUST execute the command
- AND the output MUST be displayed in the chat viewport

#### Scenario: Approve risky command
- GIVEN a command is classified as risky
- WHEN the TUI presents it with `[y/n]`
- AND the user types `y` or `yes`
- THEN the command MUST execute

#### Scenario: Deny risky command
- GIVEN a command is classified as risky
- WHEN the user types `n` or `no`
- THEN the command MUST NOT execute
- AND the agent MUST be informed the command was denied

### Requirement: User-Customizable Safe List
Users MAY add or remove command patterns via the `AllowedCommands` field in `omnisettings.json`. Custom patterns MUST be validated as valid regex at config load time. Invalid regex patterns MUST be rejected with a warning.

#### Scenario: Custom safe command
- GIVEN `omnisettings.json` contains `AllowedCommands: ["^make\\s"]`
- WHEN the agent proposes `make build`
- THEN the command MUST be classified as safe

---

## 5. State & Persistence Specification

### Purpose
JSON-based session storage enabling conversation continuity and resumption.

### Requirement: JSON History Format
The system MUST store all conversations in raw JSON format compatible with OmniGo message structures. Each session MUST be a single JSON file.

#### Scenario: Session auto-save
- GIVEN an active conversation with messages exchanged
- WHEN a new message is added (user or assistant)
- THEN the session file MUST be updated with the latest state

### Requirement: Storage Location Hierarchy
Project sessions MUST be stored in `.omni/history/session_YYYYMMDD_HHMMSS.json`. Global sessions (when not in a project) MUST be stored in `~/.config/omni/history/`. The system MUST create directories if they do not exist.

#### Scenario: Project-level storage
- GIVEN the user is in a project directory
- WHEN a session starts
- THEN history MUST be written to `.omni/history/session_YYYYMMDD_HHMMSS.json` relative to project root

#### Scenario: Global storage fallback
- GIVEN the user is NOT in a project directory
- WHEN a session starts
- THEN history MUST be written to `~/.config/omni/history/`

### Requirement: Session Resumption
The system MUST support a `--resume` CLI flag that reloads the last active session from the appropriate history directory.

#### Scenario: Resume last session
- GIVEN a previous session file exists
- WHEN the user runs `omni --resume`
- THEN the conversation history MUST be loaded from the most recent session file
- AND previous messages MUST appear in the chat viewport
- AND the agent context MUST include the full conversation history

#### Scenario: Resume with no prior session
- GIVEN no session files exist in the history directory
- WHEN the user runs `omni --resume`
- THEN the system MUST start a fresh session
- AND display a message indicating no previous session was found

### Requirement: Atomic File Writes
Session files MUST be written atomically (write to temp file, then rename) to prevent corruption on crash.

#### Scenario: Crash during save
- GIVEN a session save is in progress
- WHEN the process is interrupted mid-write
- THEN the previous valid session file MUST remain intact

---

## 6. Configuration Specification

### Purpose
Hierarchical configuration system with project-level overrides over global defaults.

### Requirement: omnisettings.json Schema
The configuration file MUST support the following fields:
- `ModelPriority`: Array of model names in fallback order (e.g., `["gpt-4o", "gemini-1.5-pro"]`)
- `AllowedCommands`: Array of regex patterns for user-approved safe commands
- `Theme`: Object containing color/style definitions for Lip Gloss/Bubble Tea components
- `McpServers`: Object mapping server names to configuration blocks (type, command, args, url, env, trusted, timeout)

#### Scenario: Valid configuration
- GIVEN a valid `omnisettings.json` file
- WHEN the application loads configuration
- THEN all fields MUST be parsed and applied

### Requirement: Resolution Hierarchy
The system MUST resolve configuration by merging project-level (`.omnisettings.json` in project root) over global-level (`~/.config/omni/omnisettings.json`). Project values MUST override global values for the same fields. Missing project config MUST fall back to global config. Missing global config MUST use built-in defaults.

#### Scenario: Project overrides global
- GIVEN global config has `ModelPriority: ["gpt-4o"]`
- AND project config has `ModelPriority: ["gemini-1.5-pro"]`
- WHEN configuration is resolved
- THEN `ModelPriority` MUST be `["gemini-1.5-pro"]`

#### Scenario: Partial project config
- GIVEN global config has `ModelPriority` and `Theme` defined
- AND project config only has `AllowedCommands` defined
- WHEN configuration is resolved
- THEN `ModelPriority` and `Theme` MUST come from global
- AND `AllowedCommands` MUST come from project

#### Scenario: No config files exist
- GIVEN neither project nor global config files exist
- WHEN the application starts
- THEN built-in defaults MUST be used
- AND the application MUST NOT error

---

## 7. Security Specification

### Purpose
Protect sensitive information from exposure in logs and output.

### Requirement: API Key Redaction
The system MUST automatically detect and mask API keys and environment variables in all log output. Patterns matching common API key formats (e.g., `sk-...`, `AIza...`, `ghp_...`) MUST be replaced with `[REDACTED]`.

#### Scenario: API key in log output
- GIVEN a log message contains `sk-abc123def456ghi789`
- WHEN the message is written to any log output
- THEN the key MUST be replaced with `[REDACTED]`

#### Scenario: Environment variable redaction
- GIVEN a command output contains `OPENAI_API_KEY=sk-abc123`
- WHEN the output is logged or displayed
- THEN the value after `=` MUST be replaced with `[REDACTED]`

### Requirement: Local-Only Data
The system MUST NOT transmit project data to any service other than the configured LLM API endpoint. Session history and configuration MUST remain strictly local.

#### Scenario: Data locality
- GIVEN a session is active
- WHEN the system communicates externally
- THEN the ONLY external endpoint contacted MUST be the LLM API
- AND no history, config, or project files SHALL be sent elsewhere
