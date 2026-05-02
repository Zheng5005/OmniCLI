# Delta for OmniCLI v1

## MODIFIED Requirements

### Requirement: Activity Indicators

The status bar MUST show the current activity state: "Searching", "Thinking", or "Executing". During any active state (Searching, Thinking, Executing, Streaming, AwaitingApproval, Wizard, McpApproval, ResourceBrowser), an animated spinner MUST be displayed alongside the activity text label. The spinner MUST start when entering an active state and stop when returning to idle. The spinner MUST NOT display during the idle/ready state.

(Previously: Status bar showed static text labels for activity states with no animation.)

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
