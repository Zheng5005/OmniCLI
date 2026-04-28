# MCP Server Lifecycle Specification

## Purpose

Server process management — spawn, health monitoring, crash recovery, and graceful SIGTERM shutdown for configured MCP servers.

## Requirements

### Requirement: Automatic Server Startup

The system MUST start all configured MCP servers during OmniCLI initialization. Each server MUST complete a handshake (`initialize` + `initialized` notification) before being marked as ready.

#### Scenario: Boot with configured servers

- GIVEN `omnisettings.json` contains 2 configured MCP servers
- WHEN OmniCLI starts
- THEN both servers MUST be spawned and complete handshake before the REPL becomes available

#### Scenario: Handshake failure

- GIVEN a configured server fails the `initialize` handshake
- WHEN the handshake times out or returns an error
- THEN the server MUST be marked as ERROR and the REPL MUST still start (non-blocking)

### Requirement: Health Monitoring

The system MUST monitor each running MCP server's health. A server is considered healthy if its process is alive and responsive. Health state MUST be reflected in the TUI status bar.

#### Scenario: Healthy server

- GIVEN a server process is running and responsive
- WHEN the health check runs
- THEN the status bar MUST show the server as "connected" or "healthy"

#### Scenario: Crashed server detection

- GIVEN a server process terminates unexpectedly
- WHEN the health monitor detects the process exit
- THEN the status bar MUST show the server as "ERROR"

### Requirement: Crash Recovery

The system MUST attempt to restart a crashed MCP server with exponential backoff. The initial retry delay MUST be 1 second, doubling up to a maximum of 30 seconds. After 5 consecutive failures, the server MUST be marked as permanently failed.

#### Scenario: Single crash recovery

- GIVEN a server crashes during an active session
- WHEN the crash is detected
- THEN the system MUST restart the server after 1 second
- AND the server MUST re-run the handshake before being marked ready

#### Scenario: Repeated crash escalation

- GIVEN a server has crashed and restarted 4 times
- WHEN it crashes a 5th time
- THEN the server MUST be marked as permanently failed
- AND no further restart attempts MUST be made until the next OmniCLI restart

### Requirement: Graceful Shutdown

The system MUST send SIGTERM to all running MCP child processes when OmniCLI exits. The system MUST wait up to 5 seconds for each process to exit before sending SIGKILL.

#### Scenario: Normal shutdown

- GIVEN 2 MCP servers are running
- WHEN the user exits OmniCLI (`/exit` or Ctrl+C)
- THEN SIGTERM MUST be sent to both child processes
- AND the system MUST wait up to 5 seconds for clean exit

#### Scenario: Forceful shutdown

- GIVEN a child process does not exit within 5 seconds of SIGTERM
- WHEN the shutdown timeout expires
- THEN SIGKILL MUST be sent to force termination

### Requirement: Server State Machine

Each MCP server MUST transition through defined states: `starting` → `ready` → `error` or `failed`. State transitions MUST emit events that the TUI and agent can subscribe to.

#### Scenario: State transition on connect

- GIVEN a server is in `starting` state
- WHEN the handshake completes successfully
- THEN the server state MUST transition to `ready`

#### Scenario: State transition on disconnect

- GIVEN a server is in `ready` state
- WHEN the process exits unexpectedly
- THEN the server state MUST transition to `error`
