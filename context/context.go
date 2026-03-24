package context

import (
	"fmt"
	"strings"

	"github.com/yordanos-habtamu/PromptOs/utils"
)

// Manager maintains a compressed code context that grows as steps execute.
// It keeps only relevant function signatures, struct definitions, and
// recent code — avoiding full file dumps to save tokens.
type Manager struct {
	segments   []string // accumulated code segments
	maxTokens  int      // approximate max characters to keep
}

// New creates a context Manager with a token budget.
func New(maxTokens int) *Manager {
	if maxTokens <= 0 {
		maxTokens = 3000
	}
	return &Manager{
		maxTokens: maxTokens,
	}
}

// Update adds new code output to the context, trimming old segments
// if the budget is exceeded.
func (m *Manager) Update(newOutput string) string {
	// Extract key parts: function signatures, struct defs, important lines
	compressed := compressCode(newOutput)
	if compressed != "" {
		m.segments = append(m.segments, compressed)
	}

	// Trim from front if over budget
	for m.totalLen() > m.maxTokens && len(m.segments) > 1 {
		m.segments = m.segments[1:]
	}

	return m.String()
}

// String returns the full current context.
func (m *Manager) String() string {
	return strings.Join(m.segments, "\n\n---\n\n")
}

// Reset clears all accumulated context.
func (m *Manager) Reset() {
	m.segments = nil
}

func (m *Manager) totalLen() int {
	total := 0
	for _, s := range m.segments {
		total += len(s)
	}
	return total
}

// compressCode extracts important lines from code output.
// Keeps: package/import, func/type signatures, struct fields, const/var.
// Skips: blank lines, pure comments, function body details.
func compressCode(code string) string {
	lines := strings.Split(code, "\n")
	var kept []string
	inBody := false
	braceDepth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Always keep: package, import, type, const, var declarations
		if strings.HasPrefix(trimmed, "package ") ||
			strings.HasPrefix(trimmed, "import") ||
			strings.HasPrefix(trimmed, "type ") ||
			strings.HasPrefix(trimmed, "const ") ||
			strings.HasPrefix(trimmed, "var ") {
			kept = append(kept, line)
			continue
		}

		// Keep function signatures
		if strings.HasPrefix(trimmed, "func ") {
			kept = append(kept, line)
			if !strings.Contains(trimmed, "{") {
				continue
			}
			inBody = true
			braceDepth = 1
			continue
		}

		// Track brace depth to skip function bodies
		if inBody {
			braceDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			if braceDepth <= 0 {
				kept = append(kept, "}")
				inBody = false
				braceDepth = 0
			}
			continue
		}

		// Keep struct fields (inside type blocks)
		if len(trimmed) > 0 && !strings.HasPrefix(trimmed, "//") {
			kept = append(kept, line)
		}
	}

	result := strings.Join(kept, "\n")
	return utils.Truncate(result, 2000)
}

// UpdateContext is a standalone function matching the signature expected by
// the executor's ExecuteAll. It creates/manages a single Manager instance.
func UpdateContext(currentCtx string, newOutput string) string {
	// Simple append-and-trim strategy for the standalone function
	combined := currentCtx
	if combined != "" {
		combined += fmt.Sprintf("\n\n--- PREVIOUS OUTPUT ---\n\n")
	}
	compressed := compressCode(newOutput)
	combined += compressed

	// Keep within reasonable bounds
	if len(combined) > 4000 {
		combined = combined[len(combined)-4000:]
	}
	return combined
}
