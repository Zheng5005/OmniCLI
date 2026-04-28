# MCP Package

This package implements the [Model Context Protocol (MCP)](https://modelcontextprotocol.io) for OmniCLI. It provides JSON-RPC 2.0 communication over stdio and SSE transports, client lifecycle management, and server health monitoring.

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Manager   │────▶│   Client     │────▶│  Transport  │
│  (server.go)│     │ (client.go)  │     │(transport.go)│
└─────────────┘     └──────────────┘     └─────────────┘
       │                     │                   │
       ▼                     ▼                   ▼
  Health Monitor      Tool/Resource        Stdio / SSE
  Crash Recovery      Discovery            JSON-RPC 2.0
```

### Key Components

- **`types.go`** — JSON-RPC 2.0 envelope types (`JSONRPCRequest`, `JSONRPCResponse`, `JSONRPCError`) and MCP protocol types (`InitializeParams`, `Tool`, `Resource`, etc.).
- **`transport.go`** — `Transport` interface with two implementations:
  - `StdioTransport` — spawns a child process and communicates over stdin/stdout using newline-delimited JSON.
  - `SSETransport` — connects to an HTTP SSE endpoint with automatic reconnection (exponential backoff, max 5 retries).
- **`client.go`** — `Client` struct that implements the MCP handshake (`Initialize`), discovers tools (`ListTools`), invokes tools (`CallTool`), and reads resources (`ListResources`, `ReadResource`).
- **`server.go`** — `Manager` struct that coordinates multiple `Client` instances. It handles lifecycle (`StartAll`, `StopAll`), health monitoring, crash recovery, and graceful shutdown.
- **`mcp_tool.go`** — `MCPTool` implements the `tools.Tool` interface. Tool names are namespace-prefixed (`server__tool`) to avoid collisions. Untrusted servers require user approval before execution.

## Usage Examples

### Starting a Manager

```go
configs := map[string]mcp.ServerConfig{
    "filesystem": {
        Type:    "stdio",
        Command: "npx",
        Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/home/user/docs"},
        Trusted: false,
    },
}

mgr := mcp.NewManager(configs)
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := mgr.StartAll(ctx); err != nil {
    log.Fatal(err)
}
defer mgr.StopAll()
```

### Calling a Tool

```go
client := mgr.Client("filesystem")
tools, err := client.ListTools(context.Background())
if err != nil {
    log.Fatal(err)
}

result, err := client.CallTool(context.Background(), "read_file", json.RawMessage(`{"path": "/home/user/docs/readme.md"}`))
```

### Creating an MCPTool

```go
tool := mcp.NewMCPTool("filesystem", toolDef, client, false, approvalFunc)
registry.Register(tool)
```

## Approval Gate

Untrusted MCP servers require explicit user approval before each tool execution. The `MCPApprovalFunc` callback receives the server name, tool name, description, and JSON arguments. Trusted servers (`Trusted: true`) bypass approval entirely.

## Health Monitoring

The `Manager` runs a health monitor goroutine per server that periodically calls `ListTools`. On failure, it attempts restart with exponential backoff. After 5 consecutive failures, the server is marked `failed` and monitoring stops.

## Graceful Shutdown

`Manager.StopAll()` sends `SIGTERM` (Unix) or `os.Interrupt` (Windows) to each child process, waits up to 5 seconds, then forcefully kills any remaining processes. SSE transports are closed cleanly and reconnection loops are terminated.
