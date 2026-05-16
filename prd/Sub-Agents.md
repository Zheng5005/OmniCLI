# Master Product Requirements Document (PRD): OmniCLI Ecosystem

**Project Name:** OmniCLI  
**Version:** 1.0.0-Master  
**Core Engine:** OmniGo AI Library (Go)  
**UI Framework:** Bubble Tea + Lip Gloss + Glamour (TUI)  

---

**Core Framework:** OmniCLI + OmniGo
**Architecture Model:** Orchestrator-Worker (Commander/Delegate)
**UI Paradigm:** Chat REPL with Inline Sub-Process Logging

--- 
## 1. Objective
To allow the primary OmniCLI agent to autonomously delegate complex, specialized sub-tasks to dedicated "Sub-Agents." Sub-agents are isolated instances configured with specific Skills, limited context windows, and financial safety boundaries, which report their findings back to the main session.

--- 
## 2. Hierarchy & Workflow Orchestration
### 2.1 The Commander-Worker Model
- **The Orchestrator (Main Agent):** Acts as the user-facing interface. It parses the user's intent, creates a plan, and determines if a task requires a specialized skill.
- **The Worker (Sub-Agent):** A temporary, background instance instantiated by the Orchestrator. Sub-agents are strictly bound to a specific **Skill JSON** (from the Skills PRD).
- **Reporting:** Sub-agents execute their tasks, summarize their output, and deliver a structured payload back to the Orchestrator before terminating.
### 2.2 Sequential Execution Flow (V1)
- **One-at-a-Time:** To ensure token predictability and UI stability, the Orchestrator will spawn and await sub-agents _sequentially_. No parallel agent pools will run in V1.
- **Autonomous Invocations:** The main agent triggers a sub-agent using a specialized structural tool call (e.g., `spawn_subagent(skill_name, sub_task_prompt, context_files)`).
---
## 3. UI/UX: The Inline Log View
The user experience transitions dynamically based on whether the agent is speaking or working:
### 3.1 Normal Chat Mode
- The interface looks like a standard Bubble Tea REPL. The user asks questions, and the main agent responds with a standard typewriter effect.
### 3.2 Sub-Agent Mode (The Log Overlay)
When the main agent calls `spawn_subagent`, the Bubble Tea UI adapts:
- **The Transition:** The input area temporarily locks. An expandable/scrollable **Log View** component renders directly below the main agent's last line.
- **Real-time Logging:** The user sees a real-time, step-by-step stream of what the sub-agent is doing without polluting the primary conversation history.
    - _Example UI Output:_
        ``` PlainText
        🤖 Orchestrator: Let me spin up a Security Expert to check this endpoint...
        📦 [Sub-Agent: security-expert] Starting instance...
        🔧 [security-expert] Calling tool: grep_search("jwt")
        🔧 [security-expert] Calling tool: read_file("internal/auth/jwt.go")
        🧠 [security-expert] Analyzing logic...
        🛑 [security-expert] Found missing validation on line 42.
        🏁 [security-expert] Task complete. Returning summary to Orchestrator.
        ```
- **Resolution:** Once the sub-agent terminates, the Log View collapses or minimizes into a status badge, and the main agent prints its final response summarizing the sub-agent's findings.
---
## 4. Context Isolation & Budget Safeguards
### 4.1 Minimal Context Snippets
To optimize performance and avoid high token overhead:
- Sub-agents do **not** inherit the entire conversation history.
- The Orchestrator extracts _only_ the relevant lines of conversation, necessary file mappings, and system constraints required to execute that specific sub-task.
### 4.2 Hard Cost Ceiling
To prevent runaway recursive loops or faulty logic from draining the user's wallet:
- **The $5 Limit:** Every sub-agent instance tracks its cumulative token spend against OmniGo’s dynamic pricing ticker.
- **Safety Brake:** If a sub-agent's internal execution hits a total cost of **$5.00 USD**, the CLI forcefully sends a cancellation signal to the context, terminates the background worker, and returns a graceful timeout error to the main Orchestrator.
---
## 5. Technical Requirements

|**Component**|**Implementation Detail**|
|---|---|
|**State Separation**|Each sub-agent runs inside its own isolated `OmniGo.Session` loop.|
|**Bubble Tea View**|Use a custom viewport or a list component styled via Lip Gloss to handle the scrolling stream of background worker logs.|
|**Inter-Agent Comms**|Workers communicate with the Orchestrator via internal structured data types (JSON payloads returned through Go channels).|

