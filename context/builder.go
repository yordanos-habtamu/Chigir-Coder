package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yordanos-habtamu/PromptOs/models"
	"github.com/yordanos-habtamu/PromptOs/utils"
)

// BuildTaskContext deterministically loads context files for a task.
// It does not accumulate prior task outputs.
func BuildTaskContext(task models.Task, projectRoot string, maxChars int) string {
	var files []string
	ctxDir := filepath.Join(projectRoot, "context")
	files = append(files,
		filepath.Join(ctxDir, "global.md"),
		filepath.Join(ctxDir, "constraints.md"),
		filepath.Join(ctxDir, fmt.Sprintf("task-%d.md", task.ID)),
	)

	var parts []string
	for _, p := range files {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("FILE: %s\n%s", filepath.Base(p), content))
	}

	if len(parts) == 0 {
		return ""
	}
	out := strings.Join(parts, "\n\n---\n\n")
	if maxChars > 0 {
		out = utils.Truncate(out, maxChars)
	}
	return out
}
