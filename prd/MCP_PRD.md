**Role:** OmniCLI acts as the **MCP Host** **Supported Transports:** Stdio (Local), SSE (Remote)

---
## 1. Functional Overview
The MCP integration allows OmniCLI to transcend its local environment by connecting to standardized external servers. This gives the agent the ability to interact with databases, web services, and third-party platforms (Slack, GitHub, etc.) using a unified protocol, without requiring new code for every tool.

---
## 2. Connectivity & Process Management
### 2.1 Transport Layer
- **Stdio (Standard I/O):** OmniCLI will spawn and manage local child processes for servers written in Python, Node.js, or Go.
- **SSE (Server-Sent Events):** OmniCLI will maintain persistent HTTP connections to remote MCP servers.
### 2.2 Lifecycle & Boot
- **Automatic Start:** Configured servers boot alongside OmniCLI.
- **Health Monitoring:** If a server process crashes, the TUI status bar will reflect an `ERROR` state for that provider.
- **Graceful Shutdown:** OmniCLI ensures all child processes receive a `SIGTERM` when the REPL is closed.
---
## 3. Configuration & Trust
### 3.1 Server Registry (`omnisettings.json`)
Servers are managed via a dedicated block. Users can define a server as "Trusted" to bypass manual approval gates.

JSON
```
{
  "mcp_servers": {
    "local-db": {
      "type": "stdio",
      "command": "python3",
      "args": ["-m", "mcp_server_postgres"],
      "trusted": true
    }
  }
}
```
### 3.2 Command Line Onboarding
- **Utility:** `omni mcp add <name> <transport-details>`
- **Action:** Performs a "handshake" test with the server before adding it to the configuration.
---
## 4. Economic Tracking (Finalized)
- **Token-Only Billing:** Costs are calculated strictly based on the tokens processed by the primary LLM (OmniGo).
- **Formula:**
    TotalCost=(TokensPrompt​+TokensMCP_Data​)×RateInput​+(TokensCompletion​)×RateOutput​
- **No External Pass-through:** Any usage-based fees from remote SSE servers are currently out of scope for the internal cost-tracker.
---
## 5. UI/UX Integration (Bubble Tea)
### 5.1 Resource Browser
- **Attachment:** A new command `/attach` allows users to pick a "Resource" from an MCP server (e.g., a DB schema) and pin it to the conversation context.
- **Visibility:** The TUI will show a "Pinned Resources" list in a collapsible side panel.
### 5.2 Tool Approval
- **Untrusted Servers:** Every MCP tool call triggers a visual confirmation box in the TUI showing the tool name and the JSON arguments.
- **Trusted Servers:** Tool calls execute silently, with a small notification in the status bar (e.g., `[✓] postgres:query`).
