# Delta for OmniCLI v1

## ADDED Requirements

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

## MODIFIED Requirements

### Requirement: omnisettings.json Schema

The configuration file MUST support the following fields:
- `ModelPriority`: Array of model names in fallback order
- `AllowedCommands`: Array of regex patterns for user-approved safe commands
- `Theme`: Object containing color/style definitions
- `McpServers`: Object mapping server names to configuration blocks (type, command, args, url, env, trusted, timeout)

(Previously: Schema had only ModelPriority, AllowedCommands, and Theme fields)

#### Scenario: Valid configuration with MCP servers

- GIVEN a valid `omnisettings.json` file with `McpServers` block
- WHEN the application loads configuration
- THEN all fields including MCP server definitions MUST be parsed and applied

#### Scenario: Partial project config

- GIVEN global config has `ModelPriority` and `McpServers` defined
- AND project config only has `AllowedCommands` defined
- WHEN configuration is resolved
- THEN `ModelPriority` and `McpServers` MUST come from global
- AND `AllowedCommands` MUST come from project

### Requirement: Bubble Tea Application Lifecycle

The application MUST implement Bubble Tea's Model interface (Init, Update, View). The program MUST initialize with a tea.Program and handle graceful shutdown on Ctrl+C or `/exit` command. On shutdown, the system MUST gracefully terminate all running MCP child processes with SIGTERM before exiting.

(Previously: Shutdown only saved session and exited — now must also terminate MCP processes)

#### Scenario: Graceful shutdown with MCP servers

- GIVEN the REPL is running with 2 MCP servers connected
- WHEN the user presses Ctrl+C or types `/exit`
- THEN the application MUST save current session
- AND send SIGTERM to all MCP child processes
- AND exit cleanly after processes terminate or timeout
