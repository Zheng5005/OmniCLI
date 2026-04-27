# Skill Generator Specification

## Purpose

Interactive CLI (`omni skill init`) that guides users through building a valid skill JSON file via a Bubble Tea TUI form.

## Requirements

### Requirement: Subcommand Parsing

The `omni` binary MUST support `skill init` as a subcommand, separate from the default TUI REPL.

#### Scenario: Run skill init
- GIVEN the user runs `omni skill init`
- WHEN the command is parsed
- THEN the skill generator TUI MUST launch (not the REPL)

#### Scenario: Pre-fill name with flag
- GIVEN the user runs `omni skill init --name my-skill`
- WHEN the generator launches
- THEN the name field MUST be pre-filled with `my-skill`

#### Scenario: Invalid subcommand
- GIVEN the user runs `omni skill foo`
- WHEN the command is parsed
- THEN an error MUST be printed: `"Unknown skill subcommand: foo"`
- AND the process MUST exit with code 1

### Requirement: Multi-Step Form

The generator MUST present a sequential form with the following steps:

| Step | Input | Validation |
|------|-------|------------|
| 1 | Skill name (kebab-case) + display name | Name matches `[a-z][a-z0-9-]*` |
| 2 | Tool selection (multi-select from registry) | At least 1 tool selected |
| 3 | System prompt (textarea) | Non-empty |
| 4 | Variable descriptions (auto-detected from prompt) | All detected variables have descriptions |
| 5 | Safe list commands (comma-separated) | Each entry is valid regex |
| 6 | Toggle `auto_execute_safe` (yes/no) | N/A |
| 7 | JSON preview + save location (project/global) | File does not already exist, or user confirms overwrite |

#### Scenario: Navigate form forward
- GIVEN the user is on step 1 and has entered valid name/display_name
- WHEN the user presses Enter
- THEN the form advances to step 2

#### Scenario: Navigate form backward
- GIVEN the user is on step 3
- WHEN the user presses Back/Escape (not cancel)
- THEN the form returns to step 2 with previous values preserved

#### Scenario: Validation blocks advance
- GIVEN the user is on step 1 with an invalid name `"My Skill!"`
- WHEN the user presses Enter
- THEN the form MUST NOT advance
- AND an error message MUST display: `"Name must be lowercase-kebab-case (e.g., docs-expert)"`

#### Scenario: Cancel form
- GIVEN the user is on any step
- WHEN the user presses Ctrl+C
- THEN the generator MUST exit without saving
- AND no file MUST be created

### Requirement: Auto-Detect Variables

After the user enters a system prompt (step 3), the generator MUST scan for `{{...}}` patterns and prompt for a description for each.

#### Scenario: Variables detected from prompt
- GIVEN the user enters `"You are a {{role}} for {{project}}."`
- WHEN step 4 loads
- THEN two input fields MUST appear: one for `role` description, one for `project` description

#### Scenario: No variables in prompt
- GIVEN the user enters a prompt with no `{{...}}` placeholders
- WHEN step 4 loads
- THEN the step MUST be skipped automatically (no variable input needed)

### Requirement: JSON Preview

Before saving, the generator MUST display the complete JSON that will be written.

#### Scenario: Preview shows complete JSON
- GIVEN all form fields are filled
- WHEN step 7 loads
- THEN the viewport MUST display the full JSON with proper indentation
- AND the user MUST be able to review before confirming

### Requirement: File Save

The generated skill JSON MUST be written to the user-selected location.

#### Scenario: Save to project directory
- GIVEN the user selects "project" as save location
- AND `.omnicli/skills/` does not exist
- WHEN the user confirms save
- THEN the directory `.omnicli/skills/` MUST be created
- AND the file `.omnicli/skills/{name}.json` MUST be written
- AND a success message MUST display with the file path

#### Scenario: Save to global directory
- GIVEN the user selects "global" as save location
- WHEN the user confirms save
- THEN the file `~/.config/omnicli/skills/{name}.json` MUST be written

#### Scenario: File already exists
- GIVEN a skill file with the same name already exists at the target location
- WHEN the user attempts to save
- THEN a confirmation prompt MUST appear: `"File exists. Overwrite? [y/n]"`
- AND if the user declines, the generator MUST return to step 7

#### Scenario: Generated file is valid
- GIVEN a skill was saved successfully
- WHEN the skill loader reads the file
- THEN it MUST parse without errors
- AND all fields MUST match the form input
