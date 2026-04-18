// Package security provides credential redaction utilities for OmniCLI.
//
// It implements an io.Writer wrapper that intercepts sensitive data such as
// API keys, tokens, and passwords, replacing them with [REDACTED] before
// forwarding to the underlying writer.
package security

import (
	"io"
	"regexp"
)

// replacement is the text used to replace redacted content.
const replacement = "[REDACTED]"

// builtinPatterns contains regex patterns for common credential formats.
var builtinPatterns = []string{
	`sk-[a-zA-Z0-9]{20,}`,                                  // OpenAI keys
	`AIza[a-zA-Z0-9_-]{35}`,                                // Google AI keys
	`ghp_[a-zA-Z0-9]{36}`,                                  // GitHub personal access tokens
	`gh[ous]_[a-zA-Z0-9]{36}`,                               // GitHub OAuth/user/service tokens
	`(?i)(api[_-]?key|secret|token|password)\s*[=:]\s*\S+`, // Generic env var values
}

// RedactingWriter wraps an io.Writer and redacts sensitive patterns from
// all data written through it.
type RedactingWriter struct {
	inner    io.Writer
	patterns []*regexp.Regexp
}

// NewRedactingWriter creates a RedactingWriter that filters output through
// the built-in credential patterns plus any additional patterns provided.
// Extra patterns that fail to compile are silently ignored.
func NewRedactingWriter(w io.Writer, extraPatterns ...string) *RedactingWriter {
	all := make([]string, 0, len(builtinPatterns)+len(extraPatterns))
	all = append(all, builtinPatterns...)
	all = append(all, extraPatterns...)

	compiled := make([]*regexp.Regexp, 0, len(all))
	for _, p := range all {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		compiled = append(compiled, re)
	}

	return &RedactingWriter{
		inner:    w,
		patterns: compiled,
	}
}

// Write implements io.Writer. It redacts any matching patterns in p before
// writing to the underlying writer. The returned byte count reflects the
// length of the original (pre-redaction) input.
func (rw *RedactingWriter) Write(p []byte) (n int, err error) {
	clean := redactWithPatterns(string(p), rw.patterns)
	_, err = rw.inner.Write([]byte(clean))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Redact replaces all occurrences of the built-in credential patterns in s
// with [REDACTED]. This is a standalone convenience function that does not
// require a RedactingWriter.
func Redact(s string) string {
	compiled := make([]*regexp.Regexp, 0, len(builtinPatterns))
	for _, p := range builtinPatterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return redactWithPatterns(s, compiled)
}

// redactWithPatterns applies the given compiled patterns to s, replacing all
// matches with [REDACTED].
func redactWithPatterns(s string, patterns []*regexp.Regexp) string {
	result := s
	for _, re := range patterns {
		result = re.ReplaceAllString(result, replacement)
	}
	return result
}
