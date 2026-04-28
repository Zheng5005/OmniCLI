# Contributing to OmniCLI

Thank you for your interest in contributing! This document covers how to build, test, and extend OmniCLI, with a focus on MCP integration.

## Development Setup

1. **Prerequisites**
   - Go 1.22 or later
   - Node.js 18+ (for MCP server testing)

2. **Clone and build**
   ```bash
   git clone https://github.com/omnicli/omnicli.git
   cd omnicli
   go build ./...
   ```

## Testing

### Unit Tests

Unit tests cover individual packages with mocked dependencies and run quickly:

```bash
go test ./...
```

### Integration Tests

Integration tests spawn real child processes and verify end-to-end behavior. They are guarded by the `integration` build tag:

```bash
go test -tags=integration ./...
```

Or run only MCP integration tests:

```bash
go test -tags=integration -run 'TestIntegration' ./internal/mcp/...
```

Integration tests require the mock server binary to be compiled. The test suite does this automatically in `TestMain` / `init()`.

### E2E Tests

E2E tests build the full `omni` binary and exercise CLI commands:

```bash
go test -tags=integration -timeout 60s -run 'TestE2E' ./cmd/omni/...
```

### Race Detection

Always run tests with the race detector before submitting:

```bash
go test -race ./...
```

## MCP Integration Testing

### Adding a New MCP Server for Testing

1. **Install an MCP server** (example: filesystem server):
   ```bash
   npx -y @modelcontextprotocol/server-filesystem /tmp/mcp-test
   ```

2. **Add it to your local config**:
   ```bash
   ./omni mcp add filesystem --type stdio --command npx --args "-y" --args "@modelcontextprotocol/server-filesystem" --args "/tmp/mcp-test"
   ```

3. **Verify it appears in the TUI**:
   - Launch `./omni`
   - Type `/mcp` to see server status
   - The status bar shows active MCP servers

4. **Write an integration test** if the server exposes novel behavior:
   - Add a test in `internal/mcp/integration_test.go`
   - Use the `//go:build integration` tag
   - Spawn the server via `mcp.NewStdioTransport` or `mcp.NewSSETransport`
   - Verify `Initialize()` → `ListTools()` → `CallTool()` round-trips

### Test Tags

| Tag | Purpose |
|-----|---------|
| *(none)* | Fast unit tests only (no external processes) |
| `integration` | Includes integration tests with real subprocesses |

Example test header:

```go
//go:build integration

package mcp

func TestIntegrationMyFeature(t *testing.T) {
    // ...
}
```

## Troubleshooting

### "Handshake failed" when adding an MCP server

- Verify the command exists in your `$PATH`
- Check that the server responds to `initialize` on stdin with a valid `InitializeResult`
- Run with `--trusted` only if you fully trust the server

### Integration tests timeout or hang

- Ensure no zombie processes: `pkill -f mcp_echo`
- The `Manager.StopAll()` function sends `SIGTERM` and waits 5s before `SIGKILL`. If tests hang, check `transport.go` `Close()` logic
- Run with verbose output: `go test -tags=integration -v -timeout 30s ./internal/mcp/...`

### Race detector failures in MCP tests

- The `Manager` and `Client` types use `sync.RWMutex` for all shared state. If you see a race, ensure you are acquiring the correct lock before accessing maps (`clients`, `status`, `configs`)
- `baseTransport` uses `sync.Mutex` for the `pending` request map

### "transport closed" errors in E2E tests

- This usually means `StopAll()` was called while a `Send()` was in flight
- Ensure test goroutines wait for `Send()` to complete before shutting down the manager

## Code Style

- Follow standard Go conventions (gofmt, go vet)
- Every exported type, function, method, and constant must have a godoc comment starting with the name of the identifier
- Keep error messages lowercase (no trailing punctuation) unless they start a sentence
- Use `fmt.Errorf("...: %w", err)` for error wrapping
