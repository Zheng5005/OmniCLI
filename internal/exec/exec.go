package exec

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"time"
)

// Classification represents whether a command is safe or risky.
type Classification int

const (
	// Safe indicates a command matched the safe list.
	Safe Classification = iota
	// Risky indicates a command did not match the safe list.
	Risky
)

// String returns the human-readable classification label.
func (c Classification) String() string {
	if c == Safe {
		return "safe"
	}
	return "risky"
}

// Result holds the output of an executed command.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Classify determines whether a command is safe or risky by checking it
// against the provided safe patterns.
func Classify(cmd string, safePatterns []*regexp.Regexp) Classification {
	if Match(cmd, safePatterns) {
		return Safe
	}
	return Risky
}

// Run executes a shell command, capturing stdout and stderr separately.
// If timeout is greater than zero, the context is wrapped with a deadline.
func Run(ctx context.Context, cmd string, timeout time.Duration) (Result, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	c := exec.CommandContext(ctx, "sh", "-c", cmd)

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()

	result := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, err
	}

	return result, nil
}
