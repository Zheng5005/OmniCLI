package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/omnicli/omnicli/internal/agent"
	"github.com/omnicli/omnicli/internal/config"
	"github.com/omnicli/omnicli/internal/exec"
	"github.com/omnicli/omnicli/internal/history"
	"github.com/omnicli/omnicli/internal/security"
	"github.com/omnicli/omnicli/internal/tools"
	"github.com/omnicli/omnicli/internal/tui"
)

func main() {
	resume := flag.Bool("resume", false, "Resume the last active session")
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

	// Create LLM client.
	llmClient, err := agent.NewOmniGoClient(cfg.ModelPriority, true, agent.DefaultSystemPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize LLM client: %v\n", err)
		os.Exit(1)
	}

	// Build tool registry.
	registry := tools.NewRegistry()
	registry.Register(&tools.ListFilesTool{})
	registry.Register(&tools.GrepSearchTool{})
	registry.Register(&tools.ReadFileTool{})

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

	// Create agent with nil send — wired after program creation.
	agentInstance := agent.New(llmClient, registry, session, nil)

	// Build TUI model and optionally load resumed history.
	model := tui.NewModel(agentInstance)
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

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
