package executor

import "strings"

// ExtractShellCommands exposes shell extraction for callers outside the package.
func ExtractShellCommands(output string) []string {
	return extractShellCommands(output)
}

// ExecCommandAllow runs a shell command using an allowlist.
func ExecCommandAllow(cmdStr string, allowlist []string) (string, error) {
	allow := make(map[string]struct{})
	for _, c := range allowlist {
		if t := strings.TrimSpace(c); t != "" {
			allow[t] = struct{}{}
		}
	}
	return execCommand(cmdStr, allow)
}

// RequiresContent reports whether a task description likely needs file/patch output.
func RequiresContent(description string) bool {
	return stepRequiresContent(description)
}

// HasFileOrPatch reports whether output contains file or patch blocks.
func HasFileOrPatch(output string) bool {
	return hasFileOrPatch(output)
}
