package exec

import (
	"fmt"
	"regexp"
)

// defaultPatterns defines commands considered safe to run without user approval.
var defaultPatterns = []string{
	// Go
	`^go\s+test\b`,
	`^go\s+fmt\b`,
	`^go\s+build\b`,
	`^go\s+mod\s+tidy\b`,
	`^go\s+list\b`,
	// Git
	`^git\s+status\b`,
	`^git\s+diff\b`,
	`^git\s+log\b`,
	`^git\s+branch\b`,
	`^git\s+show\b`,
}

// DefaultPatterns returns the default safe pattern strings.
func DefaultPatterns() []string {
	out := make([]string, len(defaultPatterns))
	copy(out, defaultPatterns)
	return out
}

// CompilePatterns compiles regex patterns, returning the successfully compiled
// list and any compilation errors. Invalid patterns are skipped with warnings
// collected in the errors slice.
func CompilePatterns(patterns []string) ([]*regexp.Regexp, []error) {
	var compiled []*regexp.Regexp
	var errs []error
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid safe-list pattern %q: %w", p, err))
			continue
		}
		compiled = append(compiled, re)
	}
	return compiled, errs
}

// Match returns true if cmd matches any of the provided patterns.
func Match(cmd string, patterns []*regexp.Regexp) bool {
	for _, re := range patterns {
		if re.MatchString(cmd) {
			return true
		}
	}
	return false
}
