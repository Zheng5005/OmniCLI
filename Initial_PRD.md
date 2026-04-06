# Product Requirements Document: OmniCLI Agent

**Project Name:** OmniCLI  
**Version:** 1.0.0-Draft  
**Engine:** Powered by OmniGo AI Library  
**UI Framework:** Bubble Tea (TUI)

---

## 1. Executive Summary
**OmniCLI** is a terminal-based AI agent designed for deep integration with software projects. It provides a "Project-Aware" REPL experience, allowing developers to interact with their codebase through a sophisticated TUI. It prioritizes speed, cost-efficiency via local search, and safety through a validated command execution system.

---

## 2. Interaction Model & UI

### 2.1 The Bubble Tea REPL
* **Architecture:** Built using the **Bubble Tea** (The Elm Architecture) framework for a stateful, interactive terminal experience.
* **Streaming:** Real-time text delivery using Go channels, rendered with a **standard typewriter effect**.
* **Markdown Rendering:** Full support for rendering Markdown (syntax-highlighted code blocks, tables, bold/italics) directly in the terminal using `Glamour`.

### 2.2 Status & Feedback
* **Global Status Bar:** A persistent footer providing:
    * **Active Model:** Current model in use (e.g., GPT-4o, Gemini 1.5 Pro).
    * **Live Session Cost:** Running USD total fetched via OmniGo’s dynamic pricing engine.
    * **Activity Indicators:** Visual cues for "Searching," "Thinking," or "Executing."

---

## 3. Project Awareness (Search-on-Demand)

### 3.1 Keyword & Regex Search
To maintain low latency and minimize token costs, OmniCLI uses a "Lazy Loading" context strategy instead of full-file ingestion.
* **Internal Tools:**
    * `list_files`: Generates a project tree (respecting `.omniignore`).
    * `grep_search`: Performs **Keyword and Regex-based searches** locally to identify relevant files.
    * `read_file`: Ingests specific lines or files into the LLM context only when the search confirms their relevance.
* **Benefit:** Efficiently handles large monorepos without exceeding context windows.

---

## 4. Command Execution & Safety

### 4.1 The "Safe List" System
OmniCLI can suggest and execute shell commands with a tiered permission model.
* **Default Safe List:** Includes common, non-destructive commands:
    * **Go:** `test`, `fmt`, `build`, `mod tidy`, `list`.
    * **Git:** `status`, `diff`, `log`, `branch`, `show`.
* **User Customization:** Users can add or delete patterns in the `omnisettings.json` file to suit their specific workflow.
* **Approval Gates:**
    * **Safe Commands:** Require a simple `[Enter]` to confirm and run.
    * **Risky/Unrecognized Commands:** Require an explicit `[y/n]` or typing `yes` to execute.

---

## 5. State & Persistence

### 5.1 JSON History
* **Storage:** All conversations are stored in **Raw JSON** format for perfect compatibility with the OmniGo library.
* **Location Hierarchy:**
    1. **Project:** `.omni/history/session_YYYYMMDD.json` (Commit-ready).
    2. **Global:** `~/.config/omni/history/` (General assistant use).
* **Session Resumption:** Supports a `--resume` flag to reload the state of the last active JSON session.

---

## 6. Configuration (`omnisettings.json`)

OmniCLI resolves configuration starting from the project root and falling back to the user's home directory.
* **Key Fields:**
    * `ModelPriority`: Global fallback order (e.g., `["gpt-4o", "gemini-1.5-pro"]`).
    * `AllowedCommands`: Array of regex patterns for pre-approved commands.
    * `Theme`: Color and style definitions for the Lip Gloss/Bubble Tea components.

---

## 7. Technical Specifications

| Component | Technology |
| :--- | :--- |
| **Logic Engine** | OmniGo (Go AI Library) |
| **TUI Framework** | [Bubble Tea](https://github.com/charmbracelet/bubbletea) |
| **Styling** | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| **Markdown** | [Glamour](https://github.com/charmbracelet/glamour) |
| **Search** | Standard `filepath.WalkDir` + `regexp` |
| **Execution** | `os/exec` with validation middleware |

---

## 8. Security & Compliance
* **Redaction:** Automatic masking of API keys and environment variables in all logs.
* **No Cloud Storage:** Project data and history remain strictly local unless the user chooses to commit history files to a remote repository.
