package validator

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/yordanos-habtamu/PromptOs/executor"
	"github.com/yordanos-habtamu/PromptOs/models"
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

// ValidateForTask applies general validation plus task-specific rules.
func ValidateForTask(output string, taskDescription string, goal string) (bool, string) {
	valid, reason := Validate(output)
	if !valid {
		return false, reason
	}
	if RequiresContent(taskDescription) && !HasCompleteFileOrPatch(output) {
		return false, "task requires // FILE or // PATCH output, but none was provided"
	}
	if requiresFootballTheme(goal, taskDescription) && !containsFootballKeywords(output) {
		return false, "output does not reflect football team context"
	}
	return true, ""
}

// IsFormatError identifies validation failures that should be rejected immediately.
func IsFormatError(reason string) bool {
	l := strings.ToLower(reason)
	return strings.Contains(l, "output is empty") ||
		strings.Contains(l, "no actionable blocks") ||
		strings.Contains(l, "suspiciously short")
}

// RequiresContent proxies the executor rule for content tasks.
func RequiresContent(description string) bool {
	return executor.RequiresContent(description)
}

// HasFileOrPatch proxies the executor check for content output.
func HasFileOrPatch(output string) bool {
	return executor.HasFileOrPatch(output)
}

// HasCompleteFileOrPatch ensures full blocks exist (not just markers).
func HasCompleteFileOrPatch(output string) bool {
	if len(parseFileBlocks(output)) > 0 {
		return true
	}
	if len(parsePatchBlocks(output)) > 0 {
		return true
	}
	return false
}

func requiresFootballTheme(goal string, taskDescription string) bool {
	g := strings.ToLower(goal)
	if !(strings.Contains(g, "football") || strings.Contains(g, "soccer")) {
		return false
	}
	d := strings.ToLower(taskDescription)
	return strings.Contains(d, ".html") || strings.Contains(d, "index.html")
}

func containsFootballKeywords(output string) bool {
	l := strings.ToLower(output)
	keywords := []string{"football", "soccer", "match", "stadium", "squad", "fans", "fixture", "kickoff", "league"}
	for _, k := range keywords {
		if strings.Contains(l, k) {
			return true
		}
	}
	return false
}

// Minimal parsers to detect complete blocks (avoid applier dependency).
func parseFileBlocks(output string) map[string]string {
	blocks := make(map[string]string)
	re := regexp.MustCompile(`(?ms)//\s*FILE:\s*(.+?)\n(.*?)//\s*END\s*FILE`)
	matches := re.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		path := strings.TrimSpace(m[1])
		content := m[2]
		blocks[path] = content
	}
	return blocks
}

func parsePatchBlocks(output string) []struct{} {
	re := regexp.MustCompile(`(?ms)//\s*PATCH:\s*(.+?)\n//\s*SEARCH:\n(.*?)//\s*REPLACE:\n(.*?)//\s*END\s*PATCH`)
	matches := re.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return nil
	}
	return make([]struct{}, len(matches))
}

// ValidateWithAI asks the model to verify the output against the task.
// Returns (valid, reason) where reason is present only on invalid.
func ValidateWithAI(client *models.Client, output string) (bool, string, error) {
	if client == nil {
		return true, "", nil
	}
	resp, err := client.Chat(prompts.ValidatorSystem, prompts.ValidatorPrompt(output))
	if err != nil {
		return false, "", err
	}
	return parseAIValidation(resp)
}

func parseAIValidation(resp string) (bool, string, error) {
	trimmed := strings.TrimSpace(resp)
	if strings.HasPrefix(strings.ToUpper(trimmed), "VALID") {
		return true, "", nil
	}
	if strings.HasPrefix(strings.ToUpper(trimmed), "INVALID") {
		parts := strings.SplitN(trimmed, ":", 2)
		reason := "AI validator reported INVALID"
		if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
			reason = strings.TrimSpace(parts[1])
		}
		return false, reason, nil
	}
	return false, "AI validator returned unexpected response", nil
}

// RunCommands executes rule-based validation commands sequentially.
// Returns false with reason if any command fails.
func RunCommands(commands []string, allowlist []string) (bool, string) {
	for _, cmd := range commands {
		if strings.TrimSpace(cmd) == "" {
			continue
		}
		if !isAllowed(cmd, allowlist) {
			return false, fmt.Sprintf("validation command not allowed: %s", cmd)
		}
		c := exec.Command("bash", "-lc", cmd)
		out, err := c.CombinedOutput()
		if err != nil {
			return false, fmt.Sprintf("validation failed: %s\n%s", cmd, strings.TrimSpace(string(out)))
		}
	}
	return true, ""
}

func isAllowed(cmdStr string, allowlist []string) bool {
	if len(allowlist) == 0 {
		return true
	}
	allow := make(map[string]struct{}, len(allowlist))
	for _, c := range allowlist {
		if t := strings.TrimSpace(c); t != "" {
			allow[t] = struct{}{}
		}
	}
	fields := strings.Fields(cmdStr)
	if len(fields) == 0 {
		return false
	}
	seps := []string{"&&", "||", ";", "|"}
	parts := []string{cmdStr}
	for _, sep := range seps {
		var next []string
		for _, p := range parts {
			split := strings.Split(p, sep)
			for _, s := range split {
				if strings.TrimSpace(s) != "" {
					next = append(next, s)
				}
			}
		}
		parts = next
	}
	for _, p := range parts {
		fs := strings.Fields(strings.TrimSpace(p))
		if len(fs) == 0 {
			continue
		}
		if _, ok := allow[fs[0]]; !ok {
			return false
		}
	}
	return true
}
