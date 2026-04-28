package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/omnicli/omnicli/internal/config"
	"github.com/omnicli/omnicli/internal/mcp"
)

// stringSlice is a flag.Value that accumulates multiple flag usages and splits
// each value by whitespace, supporting both single-flag space-separated args
// and repeated flags.
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(value string) error {
	*s = append(*s, strings.Fields(value)...)
	return nil
}

func runMCP(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: omni mcp <add|list|remove> [options]")
		os.Exit(1)
	}

	switch args[0] {
	case "add":
		runMCPAdd(args[1:])
	case "list":
		runMCPList(args[1:])
	case "remove":
		runMCPRemove(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown mcp subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func runMCPAdd(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: omni mcp add <name> [options]")
		os.Exit(1)
	}

	// Extract name (first non-flag argument) so flags can appear before or after it.
	var name string
	var flagArgs []string
	for _, a := range args {
		if name == "" && !strings.HasPrefix(a, "-") {
			name = a
		} else {
			flagArgs = append(flagArgs, a)
		}
	}
	if name == "" {
		fmt.Fprintln(os.Stderr, "Usage: omni mcp add <name> [options]")
		os.Exit(1)
	}

	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	srvType := addCmd.String("type", "", "Server type: stdio or sse")
	command := addCmd.String("command", "", "Command for stdio transport")
	var argsSlice stringSlice
	addCmd.Var(&argsSlice, "args", "Arguments for stdio transport (space-separated, can be repeated)")
	url := addCmd.String("url", "", "URL for sse transport")
	trusted := addCmd.Bool("trusted", false, "Mark server as trusted")
	if err := addCmd.Parse(flagArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if *srvType == "" {
		fmt.Fprintln(os.Stderr, "Error: --type is required")
		os.Exit(1)
	}

	cfg := mcp.ServerConfig{
		Type:    *srvType,
		Trusted: *trusted,
	}

	switch *srvType {
	case "stdio":
		if *command == "" {
			fmt.Fprintln(os.Stderr, "Error: --command is required for stdio transport")
			os.Exit(1)
		}
		cfg.Command = *command
		cfg.Args = argsSlice
	case "sse":
		if *url == "" {
			fmt.Fprintln(os.Stderr, "Error: --url is required for sse transport")
			os.Exit(1)
		}
		cfg.URL = *url
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown transport type %q\n", *srvType)
		os.Exit(1)
	}

	// Handshake test
	if err := testMCPHandshake(name, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Handshake failed: %v\n", err)
		os.Exit(1)
	}

	// Determine config file path
	path, err := mcpConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Load existing config preserving other fields
	raw, err := loadMCPConfigRaw(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Extract current mcp_servers block
	var servers map[string]mcp.ServerConfig
	if v, ok := raw["mcp_servers"]; ok {
		if err := json.Unmarshal(v, &servers); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing mcp_servers: %v\n", err)
			os.Exit(1)
		}
	} else {
		servers = make(map[string]mcp.ServerConfig)
	}

	// Duplicate check
	if _, exists := servers[name]; exists {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Server %q already exists. Overwrite? (y/n): ", name)
		text, err := reader.ReadString('\n')
		if err != nil || !strings.HasPrefix(strings.ToLower(strings.TrimSpace(text)), "y") {
			fmt.Println("Aborted.")
			os.Exit(0)
		}
	}

	servers[name] = cfg
	serversBytes, err := json.Marshal(servers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding mcp_servers: %v\n", err)
		os.Exit(1)
	}
	raw["mcp_servers"] = serversBytes

	if err := saveMCPConfigRaw(path, raw); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("MCP server %q added successfully.\n", name)
}

func testMCPHandshake(name string, cfg mcp.ServerConfig) error {
	var transport mcp.Transport
	var err error

	switch cfg.Type {
	case "stdio":
		transport, err = mcp.NewStdioTransport(cfg.Command, cfg.Args, cfg.Env)
	case "sse":
		transport = mcp.NewSSETransport(cfg.URL)
	default:
		return fmt.Errorf("unknown transport type %q", cfg.Type)
	}
	if err != nil {
		return fmt.Errorf("create transport: %w", err)
	}
	defer transport.Close()

	client := mcp.NewClient(name, cfg, transport)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Initialize(ctx); err != nil {
		return fmt.Errorf("initialize: %w", err)
	}

	return nil
}

func runMCPList(args []string) {
	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	if err := listCmd.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.McpServers) == 0 {
		fmt.Println("No MCP servers configured.")
		return
	}

	// Attempt to connect and get real-time status.
	manager := mcp.NewManager(cfg.McpServers)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	_ = manager.StartAll(ctx)
	cancel()
	statuses := manager.Status()
	manager.StopAll()

	names := make([]string, 0, len(cfg.McpServers))
	for name := range cfg.McpServers {
		names = append(names, name)
	}
	sort.Strings(names)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tTRUSTED\tSTATUS")
	for _, name := range names {
		srv := cfg.McpServers[name]
		status := "unknown"
		if st, ok := statuses[name]; ok {
			status = st.State
		}
		trusted := "no"
		if srv.Trusted {
			trusted = "yes"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, srv.Type, trusted, status)
	}
	w.Flush()
}

func runMCPRemove(args []string) {
	removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	if err := removeCmd.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	if removeCmd.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: omni mcp remove <name>")
		os.Exit(1)
	}
	name := removeCmd.Arg(0)

	// Try project config first, then global.
	paths := []string{"omnisettings.json"}
	homeDir, err := os.UserHomeDir()
	if err == nil {
		paths = append(paths, filepath.Join(homeDir, ".config", "omni", "omnisettings.json"))
	}

	removed := false
	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		raw, err := loadMCPConfigRaw(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config %s: %v\n", path, err)
			os.Exit(1)
		}
		var servers map[string]mcp.ServerConfig
		if v, ok := raw["mcp_servers"]; ok {
			if err := json.Unmarshal(v, &servers); err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing mcp_servers in %s: %v\n", path, err)
				os.Exit(1)
			}
		} else {
			continue
		}
		if _, ok := servers[name]; !ok {
			continue
		}
		delete(servers, name)
		if len(servers) == 0 {
			delete(raw, "mcp_servers")
		} else {
			serversBytes, err := json.Marshal(servers)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding mcp_servers: %v\n", err)
				os.Exit(1)
			}
			raw["mcp_servers"] = serversBytes
		}
		if err := saveMCPConfigRaw(path, raw); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("MCP server %q removed from %s.\n", name, path)
		removed = true
		break
	}

	if !removed {
		fmt.Fprintf(os.Stderr, "MCP server %q not found in any config.\n", name)
		os.Exit(1)
	}
}

// mcpConfigPath returns the path to the active config file.
// It prefers the project-level file if it exists.
func mcpConfigPath() (string, error) {
	if _, err := os.Stat("omnisettings.json"); err == nil {
		return "omnisettings.json", nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "omni", "omnisettings.json"), nil
}

func loadMCPConfigRaw(path string) (map[string]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]json.RawMessage), nil
		}
		return nil, err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	return raw, nil
}

func saveMCPConfigRaw(path string, raw map[string]json.RawMessage) error {
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}
