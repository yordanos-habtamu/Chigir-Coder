package executor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/yordanos-habtamu/PromptOs/models"
	"github.com/yordanos-habtamu/PromptOs/prompts"
)

// Executor runs plan steps one at a time through the LLM.
type Executor struct {
	client *models.Client
	skill  string
	allow  map[string]struct{}
}

// New creates a new Executor.
func New(client *models.Client, skill string, allowlist []string) *Executor {
	allow := make(map[string]struct{})
	for _, c := range allowlist {
		if t := strings.TrimSpace(c); t != "" {
			allow[t] = struct{}{}
		}
	}
	return &Executor{client: client, skill: skill, allow: allow}
}

// ExecuteStep sends a single plan step to the LLM with full plan context
// and any existing code context.
func (e *Executor) ExecuteStep(step string, stepIndex int, fullPlan []string, codeContext string) (*models.StepResult, error) {
	userPrompt := prompts.ExecutorPrompt(step, stepIndex, fullPlan, codeContext, e.skill)
	output, err := e.client.Chat(prompts.ExecutorSystem, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("execution of step %d failed: %w", stepIndex+1, err)
	}
	if stepRequiresContent(step) && !hasFileOrPatch(output) {
		return &models.StepResult{
			StepIndex: stepIndex,
			Step:      step,
			Output:    output,
			Valid:     false,
			Fixed:     false,
		}, nil
	}

	return &models.StepResult{
		StepIndex: stepIndex,
		Step:      step,
		Output:    output,
		Valid:     true, // will be checked by validator
		Fixed:     false,
	}, nil
}

// ExecuteAll runs every step in the plan sequentially, accumulating context.
// The validate and fix functions are injected so the executor stays focused.
func (e *Executor) ExecuteAll(
	plan *models.Plan,
	initialContext string,
	validateFn func(string) (bool, string),
	fixFn func(string, string) (string, error),
	commandFixFn func(string, string, string) (string, error),
	contextUpdateFn func(string, string) string,
) ([]models.StepResult, error) {
	ctx := initialContext
	var results []models.StepResult

	for i, step := range plan.Steps {
		fmt.Printf("\n⚡ Executing step %d/%d: %s\n", i+1, len(plan.Steps), step)

		result, err := e.ExecuteStep(step, i, plan.Steps, ctx)
		if err != nil {
			return results, err
		}

		// Validate
		valid, reason := validateFn(result.Output)
		result.Valid = valid

		if !valid {
			fmt.Printf("  ⚠️  Validation failed: %s\n", reason)
			fmt.Println("  🔧 Attempting fix...")

				fixed, fixErr := fixFn(result.Output, reason)
				if fixErr != nil {
					fmt.Printf("  ❌ Fix failed: %v\n", fixErr)
				} else {
					result.Output = fixed
					result.Fixed = true

				// Re-validate after fix
						valid2, reason2 := validateFn(fixed)
						result.Valid = valid2
						if !valid2 {
							fmt.Printf("  ⚠️  Still invalid after fix: %s\n", reason2)
						} else {
							fmt.Println("  ✅ Fixed successfully")
				}
			}
		} else {
			fmt.Println("  ✅ Valid output")
		}

		// Update context
		ctx = contextUpdateFn(ctx, result.Output)
		results = append(results, *result)

		// Execute only explicit // SHELL: lines to avoid accidental reruns.
		shellCmds := extractShellCommands(result.Output)
		for _, fullCmd := range shellCmds {
			cmdType, cmdArgs := splitCmd(fullCmd)
			if cmdType == "cd" && !isCompoundCommand(fullCmd) {
				fmt.Printf("  📂  Changing directory to: %s\n", cmdArgs)
				if err := os.Chdir(cmdArgs); err != nil {
					fmt.Printf("  ❌ CD Error: %v\n", err)
				}
				continue
			}
			fmt.Printf("  🖥️  Executing Shell: %s\n", fullCmd)
			out, err := execCommand(fullCmd, e.allow)
			if err != nil {
				fmt.Printf("  ❌ Shell Error: %v\n", err)
				if commandFixFn != nil {
					fixedOutput, fixErr := commandFixFn(fullCmd, out, err.Error())
					if fixErr != nil {
						fmt.Printf("  ❌ Command fix failed: %v\n", fixErr)
					} else if strings.TrimSpace(fixedOutput) != "" {
						// Try to execute any corrective shell commands immediately
						for _, fixCmd := range extractShellCommands(fixedOutput) {
							fmt.Printf("  🛠️  Applying fix: %s\n", fixCmd)
							if _, fxErr := execCommand(fixCmd, e.allow); fxErr != nil {
								fmt.Printf("  ❌ Fix command error: %v\n", fxErr)
							}
						}
						// Save fix output as a result so applier can write patches/files
						results = append(results, models.StepResult{
							StepIndex: i,
							Step:      step,
							Output:    fixedOutput,
							Valid:     true,
							Fixed:     true,
						})
					}
				}
			} else if out != "" {
				fmt.Printf("  ✅ Shell Output: %s\n", out)
			}
		}
	}

	return results, nil
}

func execCommand(cmdStr string, allow map[string]struct{}) (string, error) {
	if strings.TrimSpace(cmdStr) == "" {
		return "", nil
	}
	cmdStr = normalizeCommand(cmdStr)
	if !isAllowed(cmdStr, allow) {
		return "", fmt.Errorf("command not allowed: %s", cmdStr)
	}
	// Use a shell to support compound commands (&&, ||, ;, pipes).
	cmd := exec.Command("bash", "-lc", cmdStr)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// extractShellCommands returns ordered commands from explicit // SHELL: lines.
func extractShellCommands(output string) []string {
	var cmds []string
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "// SHELL:") {
			cmd := strings.TrimSpace(strings.TrimPrefix(line, "// SHELL:"))
			if cmd != "" {
				cmds = append(cmds, cmd)
			}
		}
	}
	return cmds
}

func splitCmd(cmdStr string) (string, string) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func isCompoundCommand(cmdStr string) bool {
	return strings.Contains(cmdStr, "&&") ||
		strings.Contains(cmdStr, "||") ||
		strings.Contains(cmdStr, ";") ||
		strings.Contains(cmdStr, "|")
}

func isAllowed(cmdStr string, allow map[string]struct{}) bool {
	if len(allow) == 0 {
		return true
	}
	fields := strings.Fields(cmdStr)
	if len(fields) == 0 {
		return false
	}
	// If compound, check each segment's first token.
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

func normalizeCommand(cmdStr string) string {
	s := strings.TrimSpace(cmdStr)
	if s == "" {
		return s
	}
	cwd, err := os.Getwd()
	if err != nil {
		return s
	}
	base := filepath.Base(cwd)
	if base == "" || base == "." || base == string(os.PathSeparator) {
		return s
	}
	// If we're already in the target dir, drop leading "cd <dir> &&"
	cdPrefix := "cd " + base + " && "
	if strings.HasPrefix(s, cdPrefix) {
		s = strings.TrimPrefix(s, cdPrefix)
	}
	// If command tries to mkdir+cd into the current dir, rewrite to "."
	for _, p := range []string{
		"mkdir -p " + base + " && cd " + base + " && ",
		"mkdir " + base + " && cd " + base + " && ",
	} {
		if strings.HasPrefix(s, p) {
			s = "mkdir -p . && " + strings.TrimPrefix(s, p)
			break
		}
	}
	return s
}

func hasFileOrPatch(output string) bool {
	return strings.Contains(output, "// FILE:") || strings.Contains(output, "// PATCH:")
}

func stepRequiresContent(step string) bool {
	l := strings.ToLower(step)
	return strings.Contains(l, "create") ||
		strings.Contains(l, "implement") ||
		strings.Contains(l, "write") ||
		strings.Contains(l, "style") ||
		strings.Contains(l, "design") ||
		strings.Contains(l, "build html") ||
		strings.Contains(l, "structure") ||
		strings.Contains(l, "layout")
}
