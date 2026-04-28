//go:build !windows

package mcp

import (
	"os"
	"syscall"
)

// termSignal returns the signal used to gracefully terminate a child process.
// On Unix-like systems this is SIGTERM.
func termSignal() os.Signal {
	return syscall.SIGTERM
}
