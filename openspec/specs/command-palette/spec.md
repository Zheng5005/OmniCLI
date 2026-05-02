# Command Palette Specification

## Purpose

A discoverable overlay for listing, filtering, and executing registered slash commands without memorization.

## Requirements

### Requirement: Palette Invocation

The system MUST open the command palette when the user presses Ctrl+P in the normal (idle) input state. The palette MUST NOT open if another overlay (help, approval, resource picker, wizard) is already active.

#### Scenario: Open palette from idle state

- GIVEN the user is in the normal input state with no overlay active
- WHEN the user presses Ctrl+P
- THEN the command palette overlay MUST appear centered on screen

#### Scenario: Palette blocked by existing overlay

- GIVEN the help dialog is already open
- WHEN the user presses Ctrl+P
- THEN the command palette MUST NOT open
- AND the existing overlay MUST remain unchanged

### Requirement: Command List Display

The palette MUST display all registered slash commands with their name and description. The list MUST be populated from the slash router's registered commands at render time.

#### Scenario: Full command list visible

- GIVEN the slash router has 7+ registered commands
- WHEN the palette opens with an empty filter
- THEN all commands MUST be visible with name and description

#### Scenario: Commands sourced from router

- GIVEN a new slash command is registered at runtime
- WHEN the palette opens
- THEN the new command MUST appear in the list

### Requirement: Type-to-Filter

The palette MUST accept keystrokes as a filter query and narrow the visible command list in real-time. Matching MUST be case-insensitive against command names and descriptions.

#### Scenario: Filter by partial name

- GIVEN the palette is open with all commands visible
- WHEN the user types "mcp"
- THEN only commands whose name or description contains "mcp" MUST be shown

#### Scenario: No matches

- GIVEN the palette is open
- WHEN the user types a query matching no command
- THEN the list MUST show zero results
- AND a "No commands found" message MUST be displayed

### Requirement: Keyboard Navigation

The palette MUST support j/Down Arrow to move selection down and k/Up Arrow to move selection up. Selection MUST wrap around at list boundaries. The first item MUST be selected by default when the palette opens or when the filter changes.

#### Scenario: Navigate down with j

- GIVEN the palette shows 5 filtered commands with the first selected
- WHEN the user presses j
- THEN the second command MUST become selected

#### Scenario: Navigate up with k

- GIVEN the palette shows 5 filtered commands with the second selected
- WHEN the user presses k
- THEN the first command MUST become selected

#### Scenario: Wrap-around at boundaries

- GIVEN the last command in the list is selected
- WHEN the user presses j (down)
- THEN the first command MUST become selected

#### Scenario: Selection resets on filter change

- GIVEN the user has navigated to the 3rd item
- WHEN the user types an additional character that changes the filter
- THEN the first item of the new filtered list MUST be selected

### Requirement: Command Execution

Pressing Enter MUST inject the selected command's text (e.g., `/mcp`) into the main input area, dismiss the palette, and return focus to the normal input state. The command MUST NOT execute immediately — it is injected for the user to review and submit.

#### Scenario: Execute selected command

- GIVEN the palette is open with `/help` selected
- WHEN the user presses Enter
- THEN the input area MUST contain `/help`
- AND the palette MUST close
- AND focus MUST return to the input area

#### Scenario: Execute with no selection

- GIVEN the filter returned zero results
- WHEN the user presses Enter
- THEN the palette MUST close
- AND the input area MUST remain unchanged

### Requirement: Palette Dismissal

Pressing Esc MUST close the palette and return to the normal input state without modifying the input area.

#### Scenario: Dismiss with Esc

- GIVEN the palette is open with a filter query typed
- WHEN the user presses Esc
- THEN the palette MUST close
- AND the input area MUST remain unchanged
