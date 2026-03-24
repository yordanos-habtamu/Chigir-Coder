package applier

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yordanos-habtamu/PromptOs/workspace"
)

// Applier takes LLM output and applies it as file changes in the workspace.
type Applier struct {
	ws *workspace.Manager
}

// New creates an Applier for the given workspace.
func New(ws *workspace.Manager) *Applier {
	return &Applier{ws: ws}
}

// Apply parses the LLM output for file blocks and writes them.
// Expected format in output:
//
//	// FILE: path/to/file.go
//	<code content>
//	// END FILE
//
// If no FILE markers are found, the output is saved as output_<step>.txt.
func (a *Applier) Apply(output string, stepIndex int) ([]string, error) {
	patches := parsePatchBlocks(output)
	var written []string
	if len(patches) > 0 {
		for _, p := range patches {
			if err := a.applyPatch(p); err != nil {
				return written, err
			}
			written = append(written, p.Path)
		}
	}

	blocks := parseFileBlocks(output)

	if len(blocks) == 0 {
		if len(written) > 0 {
			return written, nil
		}
		// No file markers found — save raw output
		path := fmt.Sprintf("output_step_%d.txt", stepIndex+1)
		if err := a.ws.WriteFile(path, output); err != nil {
			return nil, err
		}
		return []string{path}, nil
	}

	for path, content := range blocks {
		if err := a.ws.WriteFile(path, content); err != nil {
			return written, fmt.Errorf("failed to write %s: %w", path, err)
		}
		written = append(written, path)
		fmt.Printf("  📝 Wrote: %s\n", path)
	}
	return written, nil
}

// parseFileBlocks extracts file path → content mappings from LLM output.
func parseFileBlocks(output string) map[string]string {
	blocks := make(map[string]string)

	// Pattern 1: // FILE: path\n...\n// END FILE
	re := regexp.MustCompile(`(?ms)//\s*FILE:\s*(.+?)\n(.*?)//\s*END\s*FILE`)
	matches := re.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		path := strings.TrimSpace(m[1])
		content := m[2]
		blocks[path] = content
	}

	// Pattern 2: ```<lang>\n// filename: path\n...\n```
	if len(blocks) == 0 {
		re2 := regexp.MustCompile("(?ms)```(?:\\w*)\\n//\\s*(?:filename|file|path):\\s*(.+?)\\n(.*?)```")
		matches2 := re2.FindAllStringSubmatch(output, -1)
		for _, m := range matches2 {
			path := strings.TrimSpace(m[1])
			content := m[2]
			blocks[path] = content
		}
	}

	// Pattern 3: Any text that looks like a path at the start of a block
	if len(blocks) == 0 {
		re3 := regexp.MustCompile(`(?ms)//\s*(?:file:)?\s*([\w\.\-/]+)\n(.*)`)
		m3 := re3.FindStringSubmatch(output)
		if len(m3) > 2 {
			blocks[strings.TrimSpace(m3[1])] = m3[2]
		}
	}

	return blocks
}

type patchBlock struct {
	Path    string
	Search  string
	Replace string
}

// parsePatchBlocks extracts patch operations from LLM output.
// Format:
//   // PATCH: path
//   // SEARCH:
//   ...
//   // REPLACE:
//   ...
//   // END PATCH
func parsePatchBlocks(output string) []patchBlock {
	var patches []patchBlock
	re := regexp.MustCompile(`(?ms)//\s*PATCH:\s*(.+?)\n//\s*SEARCH:\n(.*?)//\s*REPLACE:\n(.*?)//\s*END\s*PATCH`)
	matches := re.FindAllStringSubmatch(output, -1)
	for _, m := range matches {
		p := patchBlock{
			Path:    strings.TrimSpace(m[1]),
			Search: m[2],
			Replace: m[3],
		}
		patches = append(patches, p)
	}
	return patches
}

func (a *Applier) applyPatch(p patchBlock) error {
	if strings.TrimSpace(p.Path) == "" {
		return fmt.Errorf("patch missing file path")
	}
	current, err := a.ws.ReadFile(p.Path)
	if err != nil {
		return err
	}
	if p.Search == "" {
		return fmt.Errorf("patch search block is empty for %s", p.Path)
	}
	if !strings.Contains(current, p.Search) {
		return fmt.Errorf("patch search text not found in %s", p.Path)
	}
	updated := strings.Replace(current, p.Search, p.Replace, 1)
	return a.ws.WriteFile(p.Path, updated)
}
