# MCP Configuration Specification

## Purpose

`mcp_servers` configuration block in `omnisettings.json` schema and `omni mcp` CLI subcommands for server management.

## Requirements

### Requirement: mcp_servers Config Schema

The system MUST support an `mcp_servers` object in `omnisettings.json`. Each entry MUST be keyed by a unique server name and contain: `type` (stdio|sse), `command` (stdio only), `args` (stdio only), `url` (sse only), `env` (optional env vars), `trusted` (boolean, default false), and `timeout` (optional, default 30s).

#### Scenario: Stdio server configuration

- GIVEN `omnisettings.json` contains an `mcp_servers` entry with `type: "stdio"`, `command`, and `args`
- WHEN the config is loaded
- THEN the server MUST be recognized as a stdio-type server

#### Scenario: SSE server configuration

- GIVEN `omnisettings.json` contains an `mcp_servers` entry with `type: "sse"` and `url`
- WHEN the config is loaded
- THEN the server MUST be recognized as an SSE-type server

#### Scenario: Invalid configuration

- GIVEN an `mcp_servers` entry has `type: "stdio"` but no `command`
- WHEN the config is loaded
- THEN a validation error MUST be reported and the server MUST be skipped

#### Scenario: Config merge with mcp_servers

- GIVEN global config has `mcp_servers` with server "A"
- AND project config has `mcp_servers` with server "B"
- WHEN config is resolved
- THEN both servers "A" and "B" MUST be available (merge, not override)

### Requirement: omni mcp add CLI Command

The system MUST implement `omni mcp add <name> [options]` CLI subcommand. The command MUST perform a handshake test with the target server before adding it to the configuration file.

#### Scenario: Add stdio server via CLI

- GIVEN the user runs `omni mcp add mydb --type stdio --command python3 --args "-m mcp_server_postgres"`
- WHEN the command executes
- THEN it MUST spawn the process, run handshake, and add the entry to config on success

#### Scenario: Add SSE server via CLI

- GIVEN the user runs `omni mcp add remote-api --type sse --url http://example.com/mcp`
- WHEN the command executes
- THEN it MUST connect via SSE, run handshake, and add the entry to config on success

#### Scenario: Handshake failure during add

- GIVEN the target server fails handshake
- WHEN `omni mcp add` is run
- THEN the command MUST exit with a non-zero code and display the error
- AND the config file MUST NOT be modified

#### Scenario: Duplicate server name

- GIVEN a server with name "mydb" already exists in config
- WHEN the user runs `omni mcp add mydb ...`
- THEN the command MUST ask for confirmation to overwrite or reject with an error

### Requirement: omni mcp list CLI Command

The system MUST implement `omni mcp list` that displays all configured servers with their type, status, and trust level.

#### Scenario: List configured servers

- GIVEN 3 servers are configured
- WHEN the user runs `omni mcp list`
- THEN each server MUST be displayed with name, type, trusted status, and connection state

### Requirement: omni mcp remove CLI Command

The system MUST implement `omni mcp remove <name>` that removes a server from the configuration file.

#### Scenario: Remove a server

- GIVEN a server "mydb" exists in config
- WHEN the user runs `omni mcp remove mydb`
- THEN the entry MUST be removed from `omnisettings.json`
- AND if the server is currently running, it MUST be shut down gracefully
