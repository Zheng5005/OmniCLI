# Slash Commands Specification

## Purpose

Extensible slash command routing system that replaces the hardcoded `/exit` check with a dispatch map supporting current and future commands.

## Requirements

### Requirement: Slash Command Parser

The system MUST detect input lines beginning with `/` and parse them into a command name and optional arguments.

#### Scenario: Simple command
- GIVEN the user types `/exit`
- WHEN the parser processes the input
- THEN the command MUST be `exit` with empty arguments

#### Scenario: Command with arguments
- GIVEN the user types `/skill documentation-expert`
- WHEN the parser processes the input
- THEN the command MUST be `skill` and arguments MUST be `"documentation-expert"`

#### Scenario: Command with multiple arguments
- GIVEN the user types `/skill docs-expert --force`
- WHEN the parser processes the input
- THEN the command MUST be `skill` and arguments MUST be `"docs-expert --force"`

#### Scenario: Empty slash
- GIVEN the user types `/`
- WHEN the parser processes the input
- THEN an error MUST be returned indicating no command specified

#### Scenario: Non-slash input
- GIVEN the user types `hello world`
- WHEN the parser processes the input
- THEN it MUST NOT be treated as a slash command (normal submit)

### Requirement: Command Router Dispatch

The router MUST maintain a map of command name → handler function. Unknown commands MUST produce a user-visible error.

#### Scenario: Registered command dispatches
- GIVEN `/exit` is registered with a quit handler
- WHEN the user types `/exit`
- THEN the quit handler MUST be invoked

#### Scenario: Skill activation command
- GIVEN `/skill` is registered with an activation handler
- WHEN the user types `/skill docs-expert`
- THEN the activation handler MUST receive `"docs-expert"` as arguments

#### Scenario: Unknown command
- GIVEN `/foo` is not registered
- WHEN the user types `/foo`
- THEN a system message MUST appear in the viewport: `"Unknown command: /foo"`
- AND the input MUST remain active (no state change)

### Requirement: Built-in Commands

The system MUST register these commands at startup:

| Command | Handler Action |
|---------|---------------|
| `/exit` | Quit the application, save session |
| `/skill <name>` | Activate the named skill |
| `/skills` | List all available skills in viewport |
| `/help` | Show available commands and usage |

#### Scenario: /skills lists available skills
- GIVEN 3 skills are available (2 project, 1 global)
- WHEN the user types `/skills`
- THEN the viewport MUST display a formatted list with name, display_name, and source for each

#### Scenario: /help shows command reference
- GIVEN the REPL is running
- WHEN the user types `/help`
- THEN the viewport MUST display a table of all registered commands with brief descriptions

#### Scenario: /skill with no argument
- GIVEN the user types `/skill` with no name
- THEN an error message MUST display: `"Usage: /skill <name>. Use /skills to list available skills."`

### Requirement: Execution Order

Slash command detection MUST run BEFORE normal message submission in the `Update` function.

#### Scenario: Slash command intercepts submit
- GIVEN the user types `/exit` and presses Enter
- WHEN the Update function processes the input
- THEN the slash router MUST handle it (quit)
- AND the input MUST NOT be sent to the LLM as a normal message

#### Scenario: Normal text passes through
- GIVEN the user types `Explain this code` and presses Enter
- WHEN the Update function processes the input
- THEN the slash router MUST NOT intercept it
- AND the text MUST be sent to the agent as a normal message

### Requirement: Handler Registration API

The router MUST expose a `Register(command string, handler SlashHandler, description string)` method for adding commands at runtime.

#### Scenario: Register new command
- GIVEN a running application
- WHEN `Register("echo", handler, "Echo back arguments")` is called
- THEN `/echo` MUST be available as a slash command
- AND it MUST appear in `/help` output

#### Scenario: Register duplicate command
- GIVEN `/exit` is already registered
- WHEN `Register("exit", newHandler, "...")` is called
- THEN the new handler MUST overwrite the existing one
- AND a warning SHOULD be logged
