# MCP Tools Specification

## Purpose

Dynamic tool discovery from MCP servers (`tools/list`), registration into the agent's tool registry, and tool execution delegation (`tools/call`).

## Requirements

### Requirement: Tool Discovery

The system MUST call `tools/list` on each MCP server after successful handshake. Discovered tools MUST be registered in the agent's tool registry with their name, description, and input schema.

#### Scenario: Discover tools from a server

- GIVEN an MCP server completes handshake in `ready` state
- WHEN the system calls `tools/list`
- THEN all returned tools MUST be registered in the agent's tool registry
- AND each tool MUST include its name, description, and JSON Schema for input validation

#### Scenario: Empty tool list

- GIVEN an MCP server returns an empty tool list
- WHEN `tools/list` completes
- THEN no tools MUST be registered from that server
- AND the system MUST NOT error

#### Scenario: Tool name collision

- GIVEN two MCP servers each expose a tool with the same name
- WHEN both servers are registered
- THEN tool names MUST be prefixed with the server name (e.g., `postgres_query`, `slack_send_message`)

### Requirement: Tool Execution Delegation

The system MUST delegate MCP tool calls to the originating server via `tools/call`. The call MUST include the tool name and arguments as defined by the tool's input schema. Results MUST be returned to the agent loop as text content.

#### Scenario: Execute MCP tool

- GIVEN the agent requests execution of `postgres_query` with args `{"sql": "SELECT 1"}`
- WHEN the system delegates to the MCP server
- THEN it MUST send `tools/call` with the tool name and arguments
- AND the server's response MUST be returned to the agent as the tool result

#### Scenario: Tool execution error

- GIVEN an MCP tool call fails on the server side
- WHEN the server returns an error response
- THEN the error message MUST be returned to the agent as the tool result (not crash the agent)

#### Scenario: Server unavailable during tool call

- GIVEN the originating MCP server is in `error` or `failed` state
- WHEN the agent attempts to call one of its tools
- THEN the system MUST return an error indicating the server is unavailable

### Requirement: Dynamic Tool Registration and Deregistration

The system MUST support adding and removing tools from the agent's tool registry at runtime. When an MCP server connects or disconnects, the tool registry MUST update and trigger session recreation.

#### Scenario: Server connects during runtime

- GIVEN an active session with no MCP tools
- WHEN a new MCP server connects and discovers tools
- THEN the tools MUST be added to the registry
- AND the OmniGo session MUST be recreated with the updated tool list

#### Scenario: Server disconnects during runtime

- GIVEN an active session with MCP tools registered
- WHEN the originating server disconnects
- THEN its tools MUST be removed from the registry
- AND the OmniGo session MUST be recreated without those tools

### Requirement: Tool Schema Compatibility

The system MUST validate that discovered tool schemas are compatible with the LLM's tool calling format. Tools with unsupported schema features MUST be logged as warnings and skipped.

#### Scenario: Valid tool schema

- GIVEN a tool with a standard JSON Schema (object type, properties, required fields)
- WHEN the tool is registered
- THEN it MUST be accepted and available for the agent to call

#### Scenario: Unsupported schema feature

- GIVEN a tool schema uses `$ref` or `oneOf` not supported by the LLM
- WHEN the tool is discovered
- THEN it MUST be skipped with a warning log
- AND the tool MUST NOT appear in the agent's available tools
