package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yordanos-habtamu/PromptOs/utils"
)

// CompileResult holds the structured prompt and extracted metadata.
type CompileResult struct {
	CleanInput string
	Keywords   []string
	Prompt     string
}

// Compile takes raw user input, normalizes it, extracts keywords,
// and builds a structured prompt ready for the planner.
func Compile(rawInput string, projectPath string) *CompileResult {
	clean := strings.TrimSpace(rawInput)
	keywords := utils.ExtractKeywords(clean)

	// Check for missing init files
	initHint := ""
	if _, err := os.Stat(filepath.Join(projectPath, "go.mod")); os.IsNotExist(err) && strings.Contains(strings.ToLower(clean), "go") {
		initHint = "MISSING go.mod. STEP 1 must be: go mod init <name>."
	}
	if _, err := os.Stat(filepath.Join(projectPath, "package.json")); os.IsNotExist(err) && strings.Contains(strings.ToLower(clean), "node") {
		initHint = "MISSING package.json. STEP 1 must be: npm init -y."
	}

	prompt := fmt.Sprintf(`TASK:%s
HINT:%s
Numbered list ONLY.`, clean, initHint)

	return &CompileResult{
		CleanInput: clean,
		Keywords:   keywords,
		Prompt:     prompt,
	}
}
