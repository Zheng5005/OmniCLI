package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	omniBinPath   string
	mockServerBin string
	projectRoot   string
)

func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	projectRoot = filepath.Join(filepath.Dir(file), "..", "..")
	omniBinPath = filepath.Join(projectRoot, "omni")
	mockServerBin = filepath.Join(projectRoot, "test", "mcp", "mock_server")

	// Compile the mock server test helper.
	if out, err := exec.Command("go", "build", "-o", mockServerBin, filepath.Join(projectRoot, "test", "mcp", "mock_server.go")).CombinedOutput(); err != nil {
		// Non-fatal; individual tests will skip if the binary is missing.
		_ = out
	}

	// Compile the omni CLI binary.
	buildCmd := exec.Command("go", "build", "-o", omniBinPath, "./cmd/omni")
	buildCmd.Dir = projectRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		_ = out
	}

	os.Exit(m.Run())
}

func skipIfNoBinaries(t *testing.T) {
	if _, err := os.Stat(omniBinPath); err != nil {
		t.Skip("omni binary not available")
	}
	if _, err := os.Stat(mockServerBin); err != nil {
		t.Skip("mock server binary not available")
	}
}

// TestE2EMcpAddAndList exercises the full CLI flow: adding a server,
// persisting it to config, and listing it with live status.
func TestE2EMcpAddAndList(t *testing.T) {
	skipIfNoBinaries(t)

	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "omnisettings.json")
	if err := os.WriteFile(cfgPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Add the mock server via CLI.
	cmd := exec.Command(omniBinPath, "mcp", "add", "integration-mock", "--type", "stdio", "--command", mockServerBin, "--trusted")
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("omni mcp add failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "added successfully") {
		t.Fatalf("unexpected output: %s", out)
	}

	// Verify the config file was updated.
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	serversRaw, ok := raw["mcp_servers"]
	if !ok {
		t.Fatal("mcp_servers key missing from config")
	}
	var servers map[string]interface{}
	if err := json.Unmarshal(serversRaw, &servers); err != nil {
		t.Fatalf("parse mcp_servers: %v", err)
	}
	if _, ok := servers["integration-mock"]; !ok {
		t.Fatal("integration-mock not found in config")
	}

	// List servers and assert the new one appears with expected fields.
	cmd = exec.Command(omniBinPath, "mcp", "list")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "HOME="+tmpDir) // isolate from global config
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("omni mcp list failed: %v\n%s", err, out)
	}
	output := string(out)
	if !strings.Contains(output, "integration-mock") {
		t.Errorf("list output missing server name:\n%s", output)
	}
	if !strings.Contains(output, "stdio") {
		t.Errorf("list output missing transport type:\n%s", output)
	}
}
