# Delta for OmniCLI v1

## ADDED Requirements

### Requirement: Skill Context Display

The TUI MUST display the currently active skill's identity in two locations: the status bar and the input prompt prefix.

#### Scenario: Status bar shows active skill
- GIVEN a skill `Docs Expert` is activated
- WHEN the status bar renders
- THEN it MUST display `SKILL: Docs Expert` between the model name and activity indicator

#### Scenario: Input prompt shows skill prefix
- GIVEN a skill `Docs Expert` is activated
- WHEN the input area renders
- THEN the prompt prefix MUST change from `> ` to `[Docs Expert] > `

#### Scenario: No skill active
- GIVEN no skill is active
- WHEN the status bar renders
- THEN no skill indicator MUST be shown
- AND the prompt prefix MUST be the default `> `

## MODIFIED Requirements

### Requirement: Bubble Tea Application Lifecycle

The application MUST implement Bubble Tea's Model interface (Init, Update, View). The program MUST initialize with a tea.Program and handle graceful shutdown on Ctrl+C or `/exit` command. The Update function MUST intercept slash commands before normal message submission. The system MUST support a `StateSkillWizard` state for variable input overlay.

(Previously: Only handled Ctrl+C and hardcoded /exit check, no slash command system or wizard state)

#### Scenario: Application startup
- GIVEN the user runs `omni` binary
- WHEN the program initializes
- THEN a Bubble Tea fullscreen app MUST launch with input area, chat viewport, and status bar
- AND the slash router MUST be initialized with built-in commands

#### Scenario: Graceful shutdown
- GIVEN the REPL is running
- WHEN the user presses Ctrl+C or types `/exit`
- THEN the application MUST save current session and exit cleanly
- AND any active skill MUST be deactivated before exit

#### Scenario: Slash command intercepts input
- GIVEN the user types `/skill docs-expert` and presses Enter
- WHEN the Update function processes the input
- THEN the slash router MUST handle the command
- AND the input MUST NOT be sent to the LLM as a normal message

#### Scenario: Skill wizard state
- GIVEN a skill with unresolved variables is activated
- WHEN the `SkillNeedsVariablesMsg` is received
- THEN the model MUST transition to `StateSkillWizard`
- AND the wizard sub-model MUST render as an overlay above the viewport
- AND normal input MUST be disabled until the wizard completes or is cancelled

### Requirement: User Input Handling

The TUI MUST provide a text input area at the bottom of the screen. The input MUST support multi-line editing. Pressing Enter (without Shift) MUST submit the message. When a skill is active, the input prompt prefix MUST display the skill's display name. When in `StateSkillWizard`, normal input is disabled and the wizard handles keyboard events.

(Previously: No skill context in prompt, no wizard state)

#### Scenario: Submit user message
- GIVEN the user has typed a prompt in the input area
- WHEN the user presses Enter
- THEN the message MUST be sent to the agent loop
- AND the input area MUST clear
- AND the message MUST appear in the chat viewport as a user message

#### Scenario: Submit with skill active
- GIVEN a skill `Docs Expert` is active
- WHEN the user types `Write API docs` and presses Enter
- THEN the message MUST be sent with the skill's resolved system prompt
- AND the prompt prefix `[Docs Expert] > ` MUST remain visible

#### Scenario: Wizard input
- GIVEN the model is in `StateSkillWizard`
- WHEN the user types in the wizard's text input and presses Enter
- THEN the current variable value MUST be captured
- AND the wizard MUST advance to the next variable (or complete if last)

#### Scenario: Cancel wizard
- GIVEN the model is in `StateSkillWizard`
- WHEN the user presses Escape
- THEN the wizard MUST cancel
- AND the model MUST return to `StateNormal`
- AND the skill MUST NOT be activated
