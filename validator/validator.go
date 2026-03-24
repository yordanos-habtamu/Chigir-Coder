package validator

import (
	"strings"

	"github.com/yordanos-habtamu/PromptOs/prompts"
	"github.com/yordanos-habtamu/PromptOs/utils"
)

// badMarkers are strings that indicate incomplete or placeholder code.
var badMarkers = []string{
	"TODO",
	"FIXME",
	"HACK",
	"XXX",
	"// ...",
	"/* ... */",
	"pass  # placeholder",
	"NotImplementedError",
	"panic(\"not implemented\")",
	"undefined",
}

// Validate checks the LLM output for signs of incompleteness or failure.
// Returns (valid bool, reason string).
func Validate(output string) (bool, string) {
	// 1. Empty or whitespace-only
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return false, "output is empty"
	}

	// 1b. Must contain at least one actionable block
	if !utils.ContainsAny(trimmed, []string{"// FILE:", "// PATCH:", "// SHELL:"}) {
		return false, "output has no actionable blocks (// FILE, // PATCH, or // SHELL)"
	}

	// 2. Too short to be real code
	if len(trimmed) < 10 {
		return false, "output is suspiciously short (< 10 chars)"
	}

	// 3. Contains bad markers
	if utils.ContainsAny(trimmed, badMarkers) {
		return false, "output contains TODO/placeholder markers"
	}

	// 4. Refusal patterns & Common Setup Errors
	for _, pattern := range prompts.RefusalPatterns {
		if strings.Contains(strings.ToLower(trimmed), strings.ToLower(pattern)) {
			return false, "output contains refusal or setup error: " + pattern
		}
	}

	return true, ""
}
