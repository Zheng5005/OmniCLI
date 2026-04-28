//go:build windows

package mcp

import "os"

// termSignal returns the signal used to gracefully terminate a child process.
// On Windows this is os.Interrupt because SIGTERM is not available.
func termSignal() os.Signal {
	return os.Interrupt
}
