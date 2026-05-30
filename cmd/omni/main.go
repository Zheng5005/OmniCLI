package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/omnicli/omnicli/internal/agent"
	"github.com/omnicli/omnicli/internal/config"
	"github.com/omnicli/omnicli/internal/exec"
	"github.com/omnicli/omnicli/internal/history"
	"github.com/omnicli/omnicli/internal/mcp"
	"github.com/omnicli/omnicli/internal/security"
	"github.com/omnicli/omnicli/internal/skills"
	"github.com/omnicli/omnicli/internal/tools"
	"github.com/omnicli/omnicli/internal/tui"
)

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "skill":
			runSkillInit(os.Args[2:])
			return
		case "mcp":
			runMCP(os.Args[2:])
			return
		}
	}

	resume := flag.Bool("resume", false, "Resume the last active session")
	skillName := flag.String("skill", "", "Activate a skill on startup")
	flag.Parse()

	// Redirect log output through the redacting writer.
	logWriter := security.NewRedactingWriter(os.Stderr)
	log.SetOutput(logWriter)

	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: config load error: %v", err)
	}

	// Determine history directory: project-local if .omni exists, else global.
	histDir := history.ProjectHistoryDir()
	if _, err := os.Stat(".omni"); os.IsNotExist(err) {
		histDir = history.GlobalHistoryDir()
	}

	// Create or resume a session.
	var session *history.Session
	if *resume {
		latestPath, err := history.Latest(histDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "No previous session found: %v\nStarting fresh session.\n", err)
			session = history.NewSession(histDir)
		} else {
			session, err = history.Load(latestPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to load session: %v\nStarting fresh session.\n", err)
				session = history.NewSession(histDir)
			}
		}
	} else {
		session = history.NewSession(histDir)
	}

	// Compile safe-list patterns for command classification.
	allPatterns := append(exec.DefaultPatterns(), cfg.AllowedCommands...)
	safePatterns, compileErrs := exec.CompilePatterns(allPatterns)
	for _, cerr := range compileErrs {
		log.Printf("Warning: %v", cerr)
	}

	// Create LLM client. If the configured models have no API key, try to
	// auto-detect from available env vars (GOOGLE_API_KEY, ANTHROPIC_API_KEY,
	// OPENAI_API_KEY). If none are set, continue without an LLM client — the
	// TUI still launches and the user is prompted only when they try to chat.
	models := agent.ResolveModels(cfg.ModelPriority)
	var llmClient agent.LLMClient
	var omniGoClient *agent.OmniGoClient
	if len(models) == 0 {
		fmt.Fprintln(os.Stderr, "Warning: no API key detected (set GOOGLE_API_KEY, ANTHROPIC_API_KEY, or OPENAI_API_KEY to enable the agent).")
	} else {
		omniGoClient, err = agent.NewOmniGoClient(models, true, agent.DefaultSystemPrompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize LLM client: %v\n", err)
		} else {
			llmClient = omniGoClient
		}
	}

	// Build tool registry.
	registry := tools.NewRegistry()
	registry.Register(&tools.ListFilesTool{})
	registry.Register(&tools.GrepSearchTool{})
	registry.Register(&tools.ReadFileTool{})

	// Initialize MCP manager with configured servers.
	mcpManager := mcp.NewManager(cfg.McpServers)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := mcpManager.StartAll(ctx); err != nil {
		log.Printf("Warning: MCP manager start error: %v", err)
	}
	cancel()
	defer func() {
		if err := mcpManager.StopAll(); err != nil {
			log.Printf("Warning: MCP manager stop error: %v", err)
		}
	}()

	// program is set after tea.NewProgram; the approval callback captures it.
	var program *tea.Program

	approvalFn := func(cmd string, classification exec.Classification) (bool, error) {
		responseCh := make(chan bool, 1)
		program.Send(tui.ApprovalRequestMsg{
			Command:        cmd,
			Classification: classification.String(),
			ResponseCh:     responseCh,
		})
		approved := <-responseCh
		return approved, nil
	}
	registry.Register(tools.NewRunCommandTool(safePatterns, approvalFn))

	// MCP approval callback. Trusted servers auto-approve; untrusted servers
	// send an MCPApprovalRequestMsg to the TUI for user confirmation.
	mcpApprovalFn := func(serverName, toolName, description, args string) (bool, error) {
		responseCh := make(chan bool, 1)
		program.Send(tui.MCPApprovalRequestMsg{
			ServerName:  serverName,
			ToolName:    toolName,
			Description: description,
			Args:        args,
			ResponseCh:  responseCh,
		})
		approved := <-responseCh
		return approved, nil
	}

	// Register MCP tools from all connected servers.
	for name, status := range mcpManager.Status() {
		if status.State != "ready" {
			continue
		}
		client := mcpManager.Client(name)
		if client == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		mcpTools, err := client.ListTools(ctx)
		cancel()
		if err != nil {
			log.Printf("Warning: failed to list tools from MCP server %s: %v", name, err)
			continue
		}
		cfg := client.Config()
		for _, toolDef := range mcpTools {
			var schemaMap map[string]any
			if len(toolDef.InputSchema) > 0 {
				if err := json.Unmarshal(toolDef.InputSchema, &schemaMap); err != nil {
					log.Printf("Warning: failed to parse schema for MCP tool %s/%s: %v", name, toolDef.Name, err)
					continue
				}
			}
			if ok, feature := mcp.ValidateSchema(schemaMap); !ok {
				log.Printf("Warning: skipping MCP tool %s/%s: unsupported schema feature %q", name, toolDef.Name, feature)
				continue
			}
			registry.Register(mcp.NewMCPTool(name, toolDef, client, cfg.Trusted, mcpApprovalFn))
		}
	}

	// Create agent with nil send — wired after program creation.
	agentInstance := agent.New(llmClient, registry, session, nil)
	agentInstance.SetClientConfig(models, true)
	agentInstance.SetMCPManager(mcpManager)
	agentInstance.SetBaseSystemPrompt(agent.DefaultSystemPrompt)

	// Recreate session with the full tool list (built-in + MCP).
	if llmClient != nil {
		allToolNames := agentInstance.AllToolNames()
		if err := agentInstance.RecreateSession(agent.DefaultSystemPrompt, allToolNames, nil); err != nil {
			log.Printf("Warning: failed to recreate session with MCP tools: %v", err)
		}
	}

	// Build slash router and skill manager.
	router := tui.NewSlashRouter()
	projectDir, globalDir := skills.DefaultDirs()
	skillManager := skills.NewManager(projectDir)
	skillManager.SetGlobalDir(globalDir)

	// Override the default /skill handler to use the real skill manager.
	router.Register("skill", func(args string) (tea.Msg, tea.Cmd) {
		name := strings.TrimSpace(args)
		if name == "" {
			return tui.SystemMsg{Content: "Usage: /skill <name>. Use /skills to list available skills."}, nil
		}
		return tui.SkillActivateMsg{Name: name}, nil
	})

	// Override /skills to list real skills.
	router.Register("skills", func(args string) (tea.Msg, tea.Cmd) {
		names, err := skillManager.List()
		if err != nil {
			return tui.SystemMsg{Content: fmt.Sprintf("Error listing skills: %v", err)}, nil
		}
		if len(names) == 0 {
			return tui.SystemMsg{Content: "No skills found. Create one with: omni skill init\n  Project skills: .omnicli/skills/*.json\n  Global skills:  ~/.config/omnicli/skills/*.json"}, nil
		}
		return tui.SystemMsg{Content: "Available skills: " + strings.Join(names, ", ")}, nil
	})

	// Build TUI model and optionally load resumed history.
	model := tui.NewModel(agentInstance, router, skillManager)
	if *resume && len(session.Messages) > 0 {
		entries := make([]tui.HistoryEntry, len(session.Messages))
		for i, m := range session.Messages {
			entries[i] = tui.HistoryEntry{Role: m.Role, Content: m.Content}
		}
		model.LoadHistory(entries)
	}

	// Create the Bubble Tea program and wire the send function.
	program = tea.NewProgram(model, tea.WithAltScreen())
	agentInstance.SetSend(func(msg interface{}) {
		program.Send(msg)
	})

	// Register spawn_subagent tool after send is wired so progress messages flow to TUI.
	if omniGoClient != nil {
		registry.Register(agent.NewSpawnSubAgentTool(omniGoClient.OmniClient(), skillManager, registry, func(msg interface{}) {
			program.Send(msg)
		}))
		allToolNames := agentInstance.AllToolNames()
		if err := agentInstance.RecreateSession(agent.DefaultSystemPrompt, allToolNames, nil); err != nil {
			log.Printf("Warning: failed to recreate session with spawn_subagent: %v", err)
		}
	}

	// Activate skill on startup if --skill flag is provided.
	if *skillName != "" {
		_, err := skillManager.Get(*skillName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skill %q not found\n", *skillName)
		} else {
			program.Send(tui.SkillActivateMsg{Name: *skillName})
		}
	}

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
