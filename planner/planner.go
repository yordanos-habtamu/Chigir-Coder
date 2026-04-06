package planner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/yordanos-habtamu/PromptOs/models"
	"github.com/yordanos-habtamu/PromptOs/prompts"
)

// Planner generates and edits execution plans via the LLM.
type Planner struct {
	client    *models.Client
	maxTokens int
}

// New creates a new Planner.
func New(client *models.Client, maxTokens int) *Planner {
	return &Planner{client: client, maxTokens: maxTokens}
}

// GeneratePlan asks the LLM to create a step-by-step plan from compiled input.
func (p *Planner) GeneratePlan(compiledPrompt string, learningHints string, skill string) (*models.Plan, error) {
	userPrompt := prompts.PlannerPrompt(compiledPrompt, learningHints, skill)
	raw, err := p.client.ChatWithOptions(prompts.PlannerSystem, userPrompt, p.maxTokens, 0)
	if err != nil {
		return nil, fmt.Errorf("plan generation failed: %w", err)
	}
	plan, err := parsePlanJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("planner returned invalid JSON plan: %w\nRaw output:\n%s", err, raw)
	}
	if len(plan.Tasks) == 0 {
		return nil, fmt.Errorf("planner returned no tasks. Raw output:\n%s", raw)
	}
	plan.Tasks = sanitizeTasks(plan.Tasks)
	return &plan, nil
}

// EditPlan writes the plan to a temp file, opens $EDITOR, and returns the
// (possibly revised) plan.
func (p *Planner) EditPlan(plan *models.Plan) (*models.Plan, error) {
	// Build plan text
	var sb strings.Builder
	sb.WriteString("# Edit this plan. Keep the numbered list format.\n")
	sb.WriteString("# Lines starting with # are ignored.\n\n")
	for i, step := range plan.Steps {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "zen-plan-*.md")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.WriteString(sb.String()); err != nil {
		tmpFile.Close()
		return nil, err
	}
	tmpFile.Close()

	// Determine editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano" // fallback
	}

	// Open editor
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("editor failed: %w", err)
	}

	// Read back
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read edited plan: %w", err)
	}

	steps := parseNumberedList(string(data))
	if len(steps) == 0 {
		// User cleared the plan → abort
		return nil, fmt.Errorf("edited plan is empty, aborting")
	}

	return &models.Plan{Steps: steps}, nil
}

// parseNumberedList extracts steps from numbered-list text.
// Handles "1. step", "1) step", "1: step" formats.
func parseNumberedList(text string) []string {
	// Standard numbered list: 1. Step
	re := regexp.MustCompile(`(?m)^\s*\d+[\.\)\:]\s*(.+)$`)
	matches := re.FindAllStringSubmatch(text, -1)
	var steps []string
	if len(matches) > 0 {
		for _, m := range matches {
			if s := strings.TrimSpace(m[1]); s != "" {
				steps = append(steps, s)
			}
		}
	}

	// Fallback: If no numbers, treat each non-empty line as a step
	if len(steps) == 0 {
		lines := strings.Split(text, "\n")
		for _, line := range lines {
			if s := strings.TrimSpace(line); s != "" && !strings.HasPrefix(s, "#") {
				steps = append(steps, s)
			}
		}
	}
	return steps
}

func sanitizePlan(steps []string) []string {
	seenMkdir := map[string]struct{}{}
	var out []string
	for _, s := range steps {
		trim := strings.TrimSpace(s)
		lower := strings.ToLower(trim)
		if dir := mkdirTarget(lower); dir != "" {
			if _, ok := seenMkdir[dir]; ok {
				continue
			}
			seenMkdir[dir] = struct{}{}
		}
		out = append(out, s)
	}
	return out
}

func sanitizeTasks(tasks []models.Task) []models.Task {
	seenMkdir := map[string]struct{}{}
	var out []models.Task
	for _, t := range tasks {
		trim := strings.TrimSpace(t.Description)
		lower := strings.ToLower(trim)
		if dir := mkdirTarget(lower); dir != "" {
			if _, ok := seenMkdir[dir]; ok {
				continue
			}
			seenMkdir[dir] = struct{}{}
		}
		out = append(out, t)
	}
	return out
}

func mkdirTarget(step string) string {
	step = strings.TrimSpace(step)
	if strings.HasPrefix(step, "mkdir -p ") {
		return strings.TrimSpace(strings.TrimPrefix(step, "mkdir -p "))
	}
	if strings.HasPrefix(step, "mkdir ") {
		return strings.TrimSpace(strings.TrimPrefix(step, "mkdir "))
	}
	return ""
}

func parsePlanJSON(text string) (models.Plan, error) {
	var plan models.Plan
	if err := json.Unmarshal([]byte(text), &plan); err != nil {
		return models.Plan{}, err
	}
	return plan, nil
}
