# MCP Transport Specification

## Purpose

JSON-RPC 2.0 protocol implementation over Stdio (local child processes) and SSE (remote HTTP) transports. Handles message framing, request/response correlation, and error handling.

## Requirements

### Requirement: JSON-RPC 2.0 Message Format

The system MUST implement JSON-RPC 2.0 compliant messages with `jsonrpc`, `method`, `params`, `id`, `result`, and `error` fields. All messages MUST be UTF-8 encoded newline-delimited JSON for Stdio transport.

#### Scenario: Request message construction

- GIVEN an MCP method call with parameters
- WHEN the transport constructs a request
- THEN the message MUST include `"jsonrpc": "2.0"`, a unique integer `id`, the `method` name, and `params` object

#### Scenario: Response correlation

- GIVEN a request was sent with `id: 1`
- WHEN a response arrives with `"id": 1`
- THEN the response MUST be routed to the original caller

#### Scenario: Error response handling

- GIVEN a server returns `{"jsonrpc": "2.0", "id": 1, "error": {"code": -32601, "message": "Method not found"}}`
- WHEN the transport processes the response
- THEN it MUST return an error to the caller with the server's error code and message

### Requirement: Stdio Transport

The system MUST spawn a child process and communicate via stdin/stdout using newline-delimited JSON. The process MUST be started with configurable command and arguments. Stderr MUST be captured separately for diagnostics.

#### Scenario: Stdio message exchange

- GIVEN a child process is running
- WHEN the host writes a JSON-RPC request to stdin followed by `\n`
- THEN the host MUST read the corresponding JSON-RPC response from stdout

#### Scenario: Stderr capture

- GIVEN a child process writes to stderr
- WHEN the transport captures stderr
- THEN the content MUST be logged for diagnostics (not treated as protocol messages)

### Requirement: SSE Transport

The system MUST establish a persistent HTTP connection to a remote MCP server using Server-Sent Events. Messages from the server MUST be parsed from SSE `data` fields. Requests to the server MUST be sent via HTTP POST to the message endpoint.

#### Scenario: SSE connection establishment

- GIVEN a remote MCP server URL
- WHEN the transport initiates an SSE connection
- THEN it MUST send an HTTP GET with `Accept: text/event-stream` header

#### Scenario: SSE message reception

- GIVEN an active SSE connection
- WHEN the server sends an event with `data: {"jsonrpc": "2.0", ...}`
- THEN the transport MUST parse the JSON and route it by `id` or as a notification

#### Scenario: SSE reconnection

- GIVEN an SSE connection drops unexpectedly
- WHEN the transport detects disconnection
- THEN it MUST attempt reconnection with exponential backoff (max 5 retries)

### Requirement: Notification Handling

The system MUST handle JSON-RPC notifications (messages without `id` field) asynchronously. Notifications MUST NOT expect a response.

#### Scenario: Server notification

- GIVEN the server sends `{"jsonrpc": "2.0", "method": "notifications/message", "params": {...}}`
- WHEN the transport receives the notification
- THEN it MUST dispatch to the appropriate handler without waiting for a response

### Requirement: Request Timeout

The system MUST enforce a configurable timeout for all JSON-RPC requests. The default timeout MUST be 30 seconds. Timed-out requests MUST return an error to the caller.

#### Scenario: Request timeout

- GIVEN a request is sent to an MCP server
- WHEN no response arrives within the configured timeout
- THEN the transport MUST return a timeout error and cancel the pending request
