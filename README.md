# OmniCLI

A terminal-based AI agent for deep codebase interaction. OmniCLI provides a project-aware REPL experience with a modern TUI, multi-model LLM support, MCP server integration, and a skill/plugin system — all from your terminal.

## Features

- **Multi-Model AI** — Supports OpenAI (GPT-4o), Anthropic (Claude), and Google (Gemini) with automatic model fallback and live cost tracking
- **Project-Aware Context** — Local search with `list_files`, `grep_search`, and `read_file` tools that respect `.omniignore`, avoiding context window overflow on large repos
- **MCP Protocol Support** — Connect to Model Context Protocol servers (stdio and SSE transports) to extend tool capabilities. Trusted servers auto-approve; untrusted servers prompt for confirmation
- **Skill System** — Load reusable skill definitions (JSON) with templated system prompts, variable wizards, and scoped tool access. Activate via `/skill <name>` or `--skill` flag
- **Sub-Agent Delegation** — Spawn isolated sub-agents with cost ceilings ($5 default) and iteration limits to handle delegated tasks in parallel
- **Command Execution Safety** — Tiered permission model with safelist patterns (git, go, etc.) and explicit approval gates for risky commands
- **Session Persistence** — JSON-based history stored per-project (`.omni/history/`) or globally (`~/.config/omni/history/`). Resume sessions with `--resume`
- **Rich TUI** — Built on Bubble Tea with streaming markdown rendering (Glamour), syntax-highlighted code blocks, a command palette (`Ctrl+P`), and resource browser (`/attach`)

## Installation

```bash
git clone https://github.com/omnicli/omnicli.git
cd omnicli
go build -o omni ./cmd/omni
```

### Prerequisites

- Go 1.24.2 or later
- At least one API key: `GOOGLE_API_KEY`, `ANTHROPIC_API_KEY`, or `OPENAI_API_KEY`

## Quick Start

```bash
# Start a new session
./omni

# Resume the last session
./omni --resume

# Activate a skill on startup
./omni --skill my-skill
```

## Usage

### Slash Commands

| Command | Description |
|---------|-------------|
| `/help` | Show available commands and shortcuts |
| `/exit` | Exit the application |
| `/mcp` | List MCP servers, status, and available tools |
| `/skills` | List available skills |
| `/skill <name>` | Activate a skill by name |
| `/attach` | Open resource browser to attach MCP resources |
| `/detach <uri>` | Detach a pinned MCP resource |

### Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+C` | Quit |
| `Ctrl+P` | Open command palette |
| `Ctrl+R` | Toggle resource panel |

### MCP Server Management

```bash
# Add a stdio server
omni mcp add filesystem --type stdio --command npx --args "-y" --args "@modelcontextprotocol/server-filesystem" --args "/path/to/dir"

# Add an SSE server
omni mcp add my-server --type sse --url http://localhost:3000/sse --trusted

# List configured servers with live status
omni mcp list

# Remove a server
omni mcp remove filesystem
```

### Skills

Skills are JSON files stored in `.omnicli/skills/` (project) or `~/.config/omnicli/skills/` (global):

```json
{
  "name": "code-review",
  "display_name": "Code Reviewer",
  "system_prompt": "You are a code reviewer. Review the changes in {{repo_path}}.",
  "variables": { "repo_path": "Path to the repository" },
  "tools": ["list_files", "read_file", "grep_search", "run_command"],
  "safe_list": ["git diff", "git log"],
  "auto_execute_safe": true
}
```

Initialize a new skill template:

```bash
omni skill init
```

## Configuration

OmniCLI resolves configuration from a hierarchy: project-level `omnisettings.json` overrides global `~/.config/omni/omnisettings.json`, with built-in defaults as fallback.

```json
{
  "modelPriority": ["gpt-4o", "claude-sonnet-4-20250514", "gemini-2.5-pro"],
  "allowedCommands": ["^npm run lint$", "^cargo check$"],
  "mcp_servers": {
    "filesystem": {
      "type": "stdio",
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"],
      "trusted": false
    }
  },
  "theme": {
    "primaryColor": "#7c3aed",
    "secondaryColor": "#6d28d9",
    "accentColor": "#a78bfa"
  }
}
```

## Architecture

```
cmd/omni/           CLI entry point, subcommands (mcp, skill)
internal/
  agent/            LLM agent loop, sub-agent delegation, tool orchestration
  config/           Configuration loading with project/global hierarchy
  exec/             Safe command execution with regex safelist
  history/          JSON session persistence and resumption
  mcp/              MCP client/manager, stdio & SSE transports
  security/         API key redaction for log output
  skills/           Skill loading, variable injection, manager
  tools/            Built-in tools: list_files, grep_search, read_file, run_command
  tui/              Bubble Tea components: model, input, viewport, status bar
```

## Development

```bash
# Run all unit tests
go test ./...

# Run integration tests (real subprocesses)
go test -tags=integration ./...

# Run with race detector
go test -race ./...

# Run E2E tests
go test -tags=integration -timeout 60s -run 'TestE2E' ./cmd/omni/...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed testing guides and MCP integration instructions.

## License

MIT
