# MCP Resources Specification (Phase 2)

## Purpose

Resource listing (`resources/list`), reading (`resources/read`), and attachment to conversation context. Includes TUI resource browser side panel and `/attach` slash command.

## Requirements

### Requirement: Resource Discovery

The system MUST call `resources/list` on each connected MCP server. Discovered resources MUST include URI, name, description, and MIME type.

#### Scenario: List available resources

- GIVEN an MCP server is in `ready` state
- WHEN the system calls `resources/list`
- THEN all returned resources MUST be cached and available for attachment

#### Scenario: Resource templates

- GIVEN a server returns resource templates (parameterized URIs)
- WHEN resources are listed
- THEN templates MUST be stored separately and resolved when the user provides parameters

### Requirement: Resource Reading

The system MUST call `resources/read` with a resource URI to retrieve its content. Content MUST be returned as text or base64-encoded binary depending on MIME type.

#### Scenario: Read text resource

- GIVEN a resource with MIME type `text/plain` exists
- WHEN `resources/read` is called with its URI
- THEN the content MUST be returned as decoded text

#### Scenario: Read binary resource

- GIVEN a resource with MIME type `application/octet-stream` exists
- WHEN `resources/read` is called with its URI
- THEN the content MUST be returned as base64-encoded data

#### Scenario: Resource not found

- GIVEN a resource URI does not exist on the server
- WHEN `resources/read` is called
- THEN an error MUST be returned to the caller

### Requirement: Resource Attachment

The system MUST support pinning MCP resources to the current conversation context. Pinned resources MUST be prepended to the system prompt or conversation context for subsequent LLM calls.

#### Scenario: Pin a resource

- GIVEN a resource is available from an MCP server
- WHEN the user attaches it via `/attach`
- THEN the resource content MUST be included in subsequent LLM requests

#### Scenario: Multiple pinned resources

- GIVEN 3 resources are pinned
- WHEN the agent sends a request to the LLM
- THEN all 3 resource contents MUST be included in the context

#### Scenario: Unpin a resource

- GIVEN a resource is currently pinned
- WHEN the user detaches it
- THEN the resource content MUST be removed from subsequent LLM requests

### Requirement: Resource Browser Panel

The system MUST provide a collapsible side panel in the TUI showing pinned MCP resources. The panel MUST display resource name, source server, and a preview of content.

#### Scenario: Panel visibility

- GIVEN no resources are pinned
- WHEN the user opens the resource panel
- THEN the panel MUST show "No pinned resources"

#### Scenario: Panel with resources

- GIVEN 2 resources are pinned
- WHEN the user opens the resource panel
- THEN each resource MUST show its name, server source, and truncated content preview

#### Scenario: Panel collapse

- GIVEN the resource panel is open
- WHEN the user toggles collapse
- THEN the panel MUST hide and free terminal space

### Requirement: /attach Slash Command

The system MUST implement `/attach` slash command that presents a resource picker. The user MUST be able to select from available resources across all connected MCP servers.

#### Scenario: Attach via slash command

- GIVEN the user types `/attach`
- WHEN the resource picker appears
- THEN the user MUST be able to navigate and select a resource
- AND the selected resource MUST be pinned to the conversation

#### Scenario: No resources available

- GIVEN no MCP servers have resources
- WHEN the user types `/attach`
- THEN the system MUST show "No resources available from connected servers"
