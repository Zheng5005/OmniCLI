package agent

import "os"

// providerKeyForModel returns the env var name that the OmniGo library expects
// for the given model identifier. Returns "" for unknown models.
func providerKeyForModel(model string) string {
	switch model {
	case "gpt-4o", "gpt-4-turbo":
		return "OPENAI_API_KEY"
	case "claude-3-5-sonnet-latest", "claude-3-5-opus-latest":
		return "ANTHROPIC_API_KEY"
	case "gemini-1.5-pro", "gemini-1.5-flash", "gemini-3.1-flash-lite-preview":
		return "GOOGLE_API_KEY"
	}
	return ""
}

// fallbackModelForKey returns a sensible default model for each provider when
// the user only has that provider's key set.
func fallbackModelForKey(envVar string) string {
	switch envVar {
	case "GOOGLE_API_KEY":
		return "gemini-3.1-flash-lite-preview"
	case "ANTHROPIC_API_KEY":
		return "claude-3-5-sonnet-latest"
	case "OPENAI_API_KEY":
		return "gpt-4o"
	}
	return ""
}

// ResolveModels filters the preferred model list to those whose API key is
// set in the environment. If none of the preferred models are usable, it
// falls back to whichever provider key is available (Google → Anthropic →
// OpenAI). Returns an empty slice if no API key is set.
func ResolveModels(preferred []string) []string {
	var usable []string
	for _, m := range preferred {
		key := providerKeyForModel(m)
		if key != "" && os.Getenv(key) != "" {
			usable = append(usable, m)
		}
	}
	if len(usable) > 0 {
		return usable
	}

	// No preferred model has its key — fall back to any available provider.
	for _, env := range []string{"GOOGLE_API_KEY", "ANTHROPIC_API_KEY", "OPENAI_API_KEY"} {
		if os.Getenv(env) != "" {
			return []string{fallbackModelForKey(env)}
		}
	}
	return nil
}
