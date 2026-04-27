**Core Framework:** OmniCLI + OmniGo
**UI Engine:** Bubble Tea (Interactive Forms)

---
## 1. Objective
To provide a modular "Plugin" architecture for OmniCLI. Skills allow the agent to assume specialized personas with restricted toolsets, custom system prompts, and interactive variable injection, ensuring high precision for specific tasks like documentation, refactoring, or security auditing.

---
## 2. Skill Definition & Storage
### 2.1 Storage Hierarchy
1. **Project-Specific:** `.omnicli/skills/*.json` (Shared via Git).
2. **Global:** `~/.config/omnicli/skills/*.json` (Personal defaults).
### 2.2 The Skill Schema
JSON
```
{
  "name": "documentation-expert",
  "display_name": "Docs Expert",
  "system_prompt": "You are a technical writer for {{project_name}}. Focus on {{focus_area}}.",
  "variables": {
    "project_name": "Name of the current project",
    "focus_area": "e.g., API, Architecture, or User Guide"
  },
  "tools": ["grep_search", "read_file", "write_file"],
  "safe_list": ["ls", "git status"],
  "auto_execute_safe": true
}
```
---
## 3. Activation Flow & Interactive Variable Injection
### 3.1 The "Setup Wizard"
When a skill is activated (via `/skill name` or `--skill name`), the CLI performs a **Variable Check**:
1. **Detection:** The CLI scans the `system_prompt` for `{{variable_name}}` tags.
2. **Interactive Form:** If variables are found, the Bubble Tea UI interrupts the REPL with a focused input form.
3. **Prompting:** For each variable, the UI displays the description provided in the JSON (e.g., "Enter the Name of the current project").
4. **Injection:** Once submitted, the variables are injected into the system prompt before the first message is sent to the LLM.
### 3.2 Visual Identity
- **Status Bar:** Displays `SKILL: Docs Expert`.
- **Prompt Change:** Input prompt updates to `[Docs Expert] >` .
- **Theme Shift:** (Optional) The TUI border color shifts to reflect the specialized mode.
---
## 4. Scoping & Tool Permissions
### 4.1 Strict Sandboxing
- **Tools:** The agent _only_ has access to the tools explicitly listed in the skill's JSON. This prevents "hallucinated tool usage" and reduces token overhead.
- **Command SafeList:** The skill’s `safe_list` replaces the global one.
- **Auto-Execution:** Commands within this list execute immediately, allowing for fluid workflows (e.g., a "Tester" skill that runs `go test` every time it finishes a fix).
---
## 5. The Skill Generator
OmniCLI will include a `skill-gen` utility to help users build these JSONs without manual editing.
- **Command:** `omni skill init` 
- **Features:**
    - Multi-select list for available OmniGo tools.
    - Text area for the System Prompt.
    - Automatic variable detection based on `{{ }}` syntax to help define the `variables` object.
---
## 6. Technical Requirements

| **Feature**         | **Implementation Detail**                                                                                         |
| ------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **Form Logic**      | Use Bubble Tea `textinput` and `list` components for the variable wizard.                                         |
| **Prompt Template** | Standard Go `text/template` or simple regex replacement for injection.                                            |
| **Hot-Swap**        | Re-instantiates the `OmniGo.Session` with the new prompt/tools while preserving (or optionally clearing) history. |

---
## 7. Test Case: `AGENTS.md` Generator
- **Purpose:** Document all local/global skills.
- **Workflow:** Activate `/skill agents-gen` -> User prompted for `{{doc_title}}` -> Agent scans `.omnicli/skills/` -> Agent writes `AGENTS.md`.
