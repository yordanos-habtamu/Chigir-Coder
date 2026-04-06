package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yordanos-habtamu/PromptOs/applier"
	"github.com/yordanos-habtamu/PromptOs/compiler"
	"github.com/yordanos-habtamu/PromptOs/config"
	appctx "github.com/yordanos-habtamu/PromptOs/context"
	"github.com/yordanos-habtamu/PromptOs/diff"
	"github.com/yordanos-habtamu/PromptOs/executor"
	"github.com/yordanos-habtamu/PromptOs/fixer"
	"github.com/yordanos-habtamu/PromptOs/learning"
	"github.com/yordanos-habtamu/PromptOs/memory"
	"github.com/yordanos-habtamu/PromptOs/models"
	"github.com/yordanos-habtamu/PromptOs/planner"
	"github.com/yordanos-habtamu/PromptOs/skills"
	"github.com/yordanos-habtamu/PromptOs/state"
	"github.com/yordanos-habtamu/PromptOs/supervisor"
	"github.com/yordanos-habtamu/PromptOs/taskqueue"
	"github.com/yordanos-habtamu/PromptOs/validator"
	"github.com/yordanos-habtamu/PromptOs/workspace"
)

// CLI flag variables — shared across the run command
var (
	flagProvider   string
	flagBaseURL    string
	flagAPIKey     string
	flagModel      string
	flagSkill      string
	flagProject    string
	flagMaxTokens  int
	flagOutputMode string
	flagNoEdit     bool
)

// RunCmd is the main "run" command that orchestrates the full pipeline.
var RunCmd = &cobra.Command{
	Use:   "run [task description]",
	Short: "Run the zen-coder pipeline on a coding task",
	Long: `Takes your feature description, generates a plan, lets you edit it,
learns from your edits, executes each step, validates, and fixes outputs.

Examples:
  zen-coder run "Add JWT authentication"
  zen-coder run --model gpt-4o --project ./my-app "Build a REST API"
  zen-coder run --provider ollama --model llama3 "Refactor the utils package"
  ZEN_API_KEY=sk-... zen-coder run "Add logging"`,
	RunE: runPipeline,
}

func init() {
	f := RunCmd.Flags()
	f.StringVar(&flagProvider, "provider", "", "LLM provider (openrouter, openai, anthropic, nvidia, ollama, custom)")
	f.StringVar(&flagBaseURL, "base-url", "", "Override API base URL")
	f.StringVar(&flagAPIKey, "api-key", "", "API key (prefer ZEN_API_KEY env var)")
	f.StringVar(&flagModel, "model", "", "Model name (e.g. gpt-4o, claude-3-5-sonnet)")
	f.StringVar(&flagSkill, "skill", "", "Skill preset (auto, landing-page, portfolio, dashboard, marketing, docs, api, cli, refactor, tests, data-pipeline, react, next)")
	f.StringVar(&flagProject, "project", "", "Project directory path (default: current dir)")
	f.IntVar(&flagMaxTokens, "max-tokens", 0, "Max tokens per LLM call")
	f.StringVar(&flagOutputMode, "output", "", "Output mode: human or json")
	f.BoolVar(&flagNoEdit, "no-edit", false, "Skip plan editing (use generated plan as-is)")
}

func runPipeline(cmd *cobra.Command, args []string) error {
	// ── Load & merge config ──
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config error: %w", err)
	}
	cfg.ApplyOverrides(flagProvider, flagBaseURL, flagAPIKey, flagModel, flagSkill, flagProject, flagOutputMode, flagMaxTokens)

	if err := cfg.Validate(); err != nil {
		return err
	}

	jsonMode := cfg.OutputMode == "json"

	// ── Show config ──
	if !jsonMode {
		fmt.Println("⚙️  Configuration:")
		fmt.Println(cfg.Summary())
		fmt.Printf("  API Key Len: %d\n", len(cfg.APIKey))
		fmt.Println()
	}

	// ── Get user input ──
	input := strings.Join(args, " ")
	if input == "" && !jsonMode {
		fmt.Print("🧠 Describe your task:\n> ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input = scanner.Text()
		}
	}
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("no input provided")
	}

	if !jsonMode {
		fmt.Printf("\n📥 Input: %s\n", input)
	}

	// ── Resolve skill ──
	skillName := cfg.Skill
	if skillName == "" || strings.EqualFold(skillName, "auto") {
		skillName = skills.DetectSkill(input)
	}
	skillText, skillErr := skills.LoadSkill(skillName)
	if skillErr != nil && !jsonMode {
		fmt.Printf("  ⚠️  Skill load error (%s): %v\n", skillName, skillErr)
	}
	if !jsonMode && skillName != "" {
		fmt.Printf("  🧩 Skill: %s\n", skillName)
	}

	// ── Initialize components ──
	client := models.NewClient(cfg.BaseURL, cfg.APIKey, cfg.Model, cfg.MaxTokens)
	plan := planner.New(client, cfg.PlannerMaxTokens)
	exec := executor.New(client, skillText, cfg.CommandAllow, cfg.TaskTimeoutSeconds, cfg.MaxTokensPerStep)
	fix := fixer.New(client, cfg.MaxFixRetries)

	// Load persistent stores
	learnStore, _ := learning.Load()
	if learnStore == nil {
		learnStore = &learning.Store{}
	}
	memStore, _ := memory.Load()
	if memStore == nil {
		memStore = &memory.Store{}
	}

	// ── Step 1: Compile ──
	if !jsonMode {
		fmt.Println("\n🔧 Compiling prompt...")
	}
	compiled := compiler.Compile(input, cfg.ProjectPath)
	if !jsonMode {
		fmt.Printf("   Keywords: %v\n", compiled.Keywords)
	}

	// Check for similar past tasks
	similar := memStore.FindSimilar(compiled.Keywords)
	if similar != nil && !jsonMode {
		fmt.Printf("   💡 Found similar past task: %s\n", similar.Input)
	}

	// Get learning hints
	hints := learnStore.GetHints(compiled.Keywords)
	if hints != "" && !jsonMode {
		fmt.Println("   📚 Applying learned improvements")
	}

	// ── Step 2: Generate Plan ──
	if !jsonMode {
		fmt.Println("\n📋 Generating plan...")
	}
	originalPlan, err := plan.GeneratePlan(compiled.Prompt, hints, skillText)
	if err != nil {
		return fmt.Errorf("planning failed: %w", err)
	}
	// Normalize task defaults
	for i := range originalPlan.Tasks {
		if originalPlan.Tasks[i].ID == 0 {
			originalPlan.Tasks[i].ID = i + 1
		}
		if strings.TrimSpace(originalPlan.Tasks[i].Name) == "" {
			originalPlan.Tasks[i].Name = fmt.Sprintf("Task %d", i+1)
		}
		if strings.TrimSpace(originalPlan.Tasks[i].Description) == "" {
			originalPlan.Tasks[i].Description = originalPlan.Tasks[i].Name
		}
		if strings.TrimSpace(originalPlan.Tasks[i].Status) == "" {
			originalPlan.Tasks[i].Status = "pending"
		}
		if originalPlan.Tasks[i].MaxRetries == 0 {
			originalPlan.Tasks[i].MaxRetries = cfg.MaxFixRetries
		}
	}

	if !jsonMode {
		fmt.Println("\n📋 Generated Plan:")
		for i, task := range originalPlan.Tasks {
			fmt.Printf("   %d. %s — %s\n", i+1, task.Name, task.Description)
		}
	}

	// ── Step 3: Edit Plan ──
	var editedPlan *models.Plan
	if flagNoEdit || jsonMode || len(originalPlan.Tasks) > 0 {
		// Skip editing in no-edit mode or JSON mode (machine integration)
		editedPlan = originalPlan
	} else {
		fmt.Print("\n✏️  Edit plan? (y/n): ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() && strings.ToLower(strings.TrimSpace(scanner.Text())) == "y" {
			editedPlan, err = plan.EditPlan(originalPlan)
			if err != nil {
				return fmt.Errorf("plan editing failed: %w", err)
			}
			if !jsonMode {
				fmt.Println("\n📋 Revised Plan:")
				revisedSteps := tasksToSteps(editedPlan.Tasks)
				for i, step := range revisedSteps {
					fmt.Printf("   %d. %s\n", i+1, step)
				}
			}
		} else {
			editedPlan = originalPlan
		}
	}

	// ── Step 4: Diff & Learn ──
	origSteps := tasksToSteps(originalPlan.Tasks)
	editedSteps := tasksToSteps(editedPlan.Tasks)
	d := diff.DiffPlans(&models.Plan{Steps: origSteps}, &models.Plan{Steps: editedSteps})
	if d.Changed {
		if !jsonMode {
			fmt.Println("\n📊 Plan changes detected:")
			for _, a := range d.Added {
				fmt.Printf("   ✅ Added: %s\n", a)
			}
			for _, r := range d.Removed {
				fmt.Printf("   ❌ Removed: %s\n", r)
			}
		}
		newLearnings := diff.ExtractLearnings(d, input)
		learnStore.AddLearnings(newLearnings)
		if saveErr := learnStore.Save(); saveErr == nil && !jsonMode {
			fmt.Println("   💾 Learnings saved")
		}
	}

	// ── Step 5: Execute ──
	if !jsonMode {
		fmt.Println("\n🚀 Starting execution...")
	}

	// Setup workspace
	var app *applier.Applier
	ws, wsErr := workspace.New(cfg.ProjectPath)
	if wsErr == nil {
		app = applier.New(ws)
		if chErr := os.Chdir(ws.Root); chErr != nil && !jsonMode {
			fmt.Printf("  ⚠️  Could not set working dir: %v\n", chErr)
		}
		if !jsonMode {
			summary, _ := ws.GetSummary()
			fmt.Printf("\n📁 Workspace:\n%s\n", summary)
		}
	}

	// Build function references for the executor loop
	validateFn := func(output string, taskDesc string, goal string) (bool, string) {
		return validator.ValidateForTask(output, taskDesc, goal)
	}
	fixFn := func(code string, reason string, taskDesc string, requiresContent bool) (string, error) {
		return fix.Fix(code, reason, taskDesc, requiresContent)
	}
	commandFixFn := func(cmd string, output string, reason string) (string, error) {
		return fix.FixCommand(cmd, output, reason)
	}
	// Initialize state store
	stateStore, stErr := state.New(input, *editedPlan)
	if stErr == nil {
		_ = stateStore.Save()
		if !jsonMode {
			fmt.Printf("  🧾 State file: %s\n", stateStore.Path())
		}
	} else if !jsonMode {
		fmt.Printf("  ⚠️  Could not initialize state store: %v\n", stErr)
	}

	// Build deterministic task queue
	queue := taskqueue.New(editedPlan.Tasks)
	completed := map[int]bool{}
	totalTasks := len(editedPlan.Tasks)
	contextRoot := cfg.ProjectPath
	if ws != nil {
		contextRoot = ws.Root
	}
	goal := input
	if strings.TrimSpace(editedPlan.Goal) != "" {
		goal = editedPlan.Goal
	}

	var results []models.StepResult
	for {
		task := queue.Next(completed)
		if task == nil {
			break
		}
		attempt := 0
		for {
			attempt++
			task.Status = "running"
			if stateStore != nil {
				stateStore.SetCurrentTask(task.ID)
				stateStore.UpdateTask(*task)
				_ = stateStore.Save()
			}

			fmt.Printf("\n⚡ Executing task %d/%d (attempt %d): %s\n", len(completed)+1, totalTasks, attempt, task.Name)

			taskContext := appctx.BuildTaskContext(*task, contextRoot, cfg.ContextBudget)
			if strings.TrimSpace(goal) != "" {
				taskContext = "GOAL:\n" + goal + "\n\n" + taskContext
			}
			result, err := exec.ExecuteStep(*task, taskContext)
			if err != nil {
				if !jsonMode {
					fmt.Printf("  ❌ Execution error: %v\n", err)
				}
				task.Error = err.Error()
				task.Retries++
				if task.Retries > task.MaxRetries {
					decision, decErr := supervisor.Decide(client, *task, err.Error(), task.Retries, cfg.UseAISupervisor)
					if decErr != nil && !jsonMode {
						fmt.Printf("  ⚠️  Supervisor error: %v\n", decErr)
					}
					if !jsonMode {
						fmt.Printf("  🧭 Supervisor: %s (%s)\n", decision.Action, decision.Reason)
					}
					if decision.Action == "retry" {
						continue
					}
					if decision.Action == "modify" && strings.TrimSpace(decision.Modification) != "" {
						task.Description = decision.Modification
						task.Retries = 0
						continue
					}
					if decision.Action == "skip" {
						task.Status = "failed"
						if stateStore != nil {
							stateStore.MarkFailed(task.ID)
							stateStore.UpdateTask(*task)
							_ = stateStore.Save()
						}
						break
					}
					// abort default
					task.Status = "failed"
					if stateStore != nil {
						stateStore.MarkFailed(task.ID)
						stateStore.UpdateTask(*task)
						_ = stateStore.Save()
					}
					return fmt.Errorf("aborted by supervisor: %s", decision.Reason)
				}
				continue
			}

			// If model returned plain content without blocks, wrap it into a FILE block.
			if executor.RequiresContent(task.Description) && !validator.HasCompleteFileOrPatch(result.Output) {
				if wrapped := wrapContentAsFile(result.Output, task.Description); wrapped != "" {
					result.Output = wrapped
				}
			}

			valid, reason := validateFn(result.Output, task.Description, goal)
			if valid && cfg.UseAIValidator {
				aiValid, aiReason, aiErr := validator.ValidateWithAI(client, result.Output)
				if aiErr != nil && !jsonMode {
					fmt.Printf("  ⚠️  AI validator error: %v\n", aiErr)
				} else if !aiValid {
					valid = false
					reason = fmt.Sprintf("AI validator: %s", aiReason)
				}
			}
			result.Valid = valid
			if !valid {
				if validator.IsFormatError(reason) {
					if !jsonMode {
						fmt.Printf("  ❌ Format error (no fix attempt): %s\n", reason)
					}
					task.Retries++
					task.Error = reason
					if task.Retries > task.MaxRetries {
						decision, decErr := supervisor.Decide(client, *task, reason, task.Retries, cfg.UseAISupervisor)
						if decErr != nil && !jsonMode {
							fmt.Printf("  ⚠️  Supervisor error: %v\n", decErr)
						}
						if !jsonMode {
							fmt.Printf("  🧭 Supervisor: %s (%s)\n", decision.Action, decision.Reason)
						}
						if decision.Action == "retry" {
							continue
						}
						if decision.Action == "modify" && strings.TrimSpace(decision.Modification) != "" {
							task.Description = decision.Modification
							task.Retries = 0
							continue
						}
						if decision.Action == "skip" {
							task.Status = "failed"
							if stateStore != nil {
								stateStore.MarkFailed(task.ID)
								stateStore.UpdateTask(*task)
								_ = stateStore.Save()
							}
							break
						}
						task.Status = "failed"
						if stateStore != nil {
							stateStore.MarkFailed(task.ID)
							stateStore.UpdateTask(*task)
							_ = stateStore.Save()
						}
						return fmt.Errorf("aborted by supervisor: %s", decision.Reason)
					}
					continue
				}
				if !jsonMode {
					fmt.Printf("  ⚠️  Validation failed: %s\n", reason)
					fmt.Println("  🔧 Attempting fix...")
				}
				fixed, fixErr := fixFn(result.Output, reason, task.Description, executor.RequiresContent(task.Description))
				if fixErr != nil {
					if !jsonMode {
						fmt.Printf("  ❌ Fix failed: %v\n", fixErr)
					}
				} else {
					result.Output = fixed
					result.Fixed = true
					valid2, reason2 := validateFn(fixed, task.Description, goal)
					result.Valid = valid2
					if !valid2 && !jsonMode {
						fmt.Printf("  ⚠️  Still invalid after fix: %s\n", reason2)
					} else if !jsonMode {
						fmt.Println("  ✅ Fixed successfully")
					}
				}
				if !result.Valid {
					task.Retries++
					task.Error = reason
					if task.Retries > task.MaxRetries {
						decision, decErr := supervisor.Decide(client, *task, reason, task.Retries, cfg.UseAISupervisor)
						if decErr != nil && !jsonMode {
							fmt.Printf("  ⚠️  Supervisor error: %v\n", decErr)
						}
						if !jsonMode {
							fmt.Printf("  🧭 Supervisor: %s (%s)\n", decision.Action, decision.Reason)
						}
						if decision.Action == "retry" {
							continue
						}
						if decision.Action == "modify" && strings.TrimSpace(decision.Modification) != "" {
							task.Description = decision.Modification
							task.Retries = 0
							continue
						}
						if decision.Action == "skip" {
							task.Status = "failed"
							if stateStore != nil {
								stateStore.MarkFailed(task.ID)
								stateStore.UpdateTask(*task)
								_ = stateStore.Save()
							}
							break
						}
						task.Status = "failed"
						if stateStore != nil {
							stateStore.MarkFailed(task.ID)
							stateStore.UpdateTask(*task)
							_ = stateStore.Save()
						}
						return fmt.Errorf("aborted by supervisor: %s", decision.Reason)
					}
					continue
				}
			} else if !jsonMode {
				fmt.Println("  ✅ Valid output")
			}

			results = append(results, *result)

			// Execute explicit shell commands
			shellCmds := executor.ExtractShellCommands(result.Output)
			for _, fullCmd := range shellCmds {
				fmt.Printf("  🖥️  Executing Shell: %s\n", fullCmd)
				out, err := executor.ExecCommandAllow(fullCmd, cfg.CommandAllow)
				if err != nil && !jsonMode {
					fmt.Printf("  ❌ Shell Error: %v\n", err)
					if commandFixFn != nil {
						fixedOutput, fixErr := commandFixFn(fullCmd, out, err.Error())
						if fixErr != nil {
							fmt.Printf("  ❌ Command fix failed: %v\n", fixErr)
						} else if strings.TrimSpace(fixedOutput) != "" {
							for _, fixCmd := range executor.ExtractShellCommands(fixedOutput) {
								fmt.Printf("  🛠️  Applying fix: %s\n", fixCmd)
								if _, fxErr := executor.ExecCommandAllow(fixCmd, cfg.CommandAllow); fxErr != nil {
									fmt.Printf("  ❌ Fix command error: %v\n", fxErr)
								}
							}
							results = append(results, models.StepResult{
								StepIndex: task.ID,
								Step:      task.Description,
								Output:    fixedOutput,
								Valid:     true,
								Fixed:     true,
							})
						}
					}
				} else if out != "" && !jsonMode {
					fmt.Printf("  ✅ Shell Output: %s\n", out)
				}
			}

			if result.Valid {
				task.Status = "done"
				task.Output = result.Output
				completed[task.ID] = true
				if stateStore != nil {
					stateStore.MarkCompleted(task.ID)
					stateStore.UpdateTask(*task)
					_ = stateStore.Save()
				}
			} else {
				task.Status = "failed"
				task.Error = "validation failed"
				if stateStore != nil {
					stateStore.MarkFailed(task.ID)
					stateStore.UpdateTask(*task)
					_ = stateStore.Save()
				}
			}
			break
		}
	}

	// ── Step 6: Apply outputs to workspace ──
	var writtenFiles []string
	if app != nil {
		if !jsonMode {
			fmt.Println("\n📝 Applying outputs to workspace...")
		}
		for _, r := range results {
			requireContent := executor.RequiresContent(r.Step)
			files, applyErr := app.ApplyStrict(r.Output, r.StepIndex, requireContent)
			if applyErr != nil {
				if !jsonMode {
					fmt.Printf("  ⚠️  Apply error for step %d: %v\n", r.StepIndex+1, applyErr)
				}
			} else {
				writtenFiles = append(writtenFiles, files...)
				if !jsonMode {
					fmt.Printf("  ✅ Step %d → %v\n", r.StepIndex+1, files)
				}
			}
		}
	}

	// ── Step 7: Rule-based validation ──
	validationOK := true
	validationReason := ""
	if len(cfg.ValidationCommands) > 0 {
		if !jsonMode {
			fmt.Println("\n🧪 Running validation commands...")
		}
		ok, reason := validator.RunCommands(cfg.ValidationCommands, cfg.CommandAllow)
		validationOK = ok
		validationReason = reason
		if !jsonMode {
			if ok {
				fmt.Println("  ✅ Validation passed")
			} else {
				fmt.Printf("  ❌ Validation failed: %s\n", reason)
			}
		}
	}

	// ── Step 8: Save to memory ──
	memStore.AddRecord(input, origSteps, editedSteps, compiled.Keywords)
	_ = memStore.Save()

	// ── Output ──
	if jsonMode {
		// Machine-readable JSON output for Cline/Roo Code
		output := map[string]interface{}{
			"input": input,
			"config": map[string]string{
				"provider": cfg.Provider,
				"model":    cfg.Model,
				"project":  cfg.ProjectPath,
			},
			"original_plan": origSteps,
			"final_plan":    editedSteps,
			"plan":          editedPlan,
			"results":       results,
			"files_written": writtenFiles,
			"plan_changed":  d.Changed,
			"validation": map[string]interface{}{
				"ok":     validationOK,
				"reason": validationReason,
			},
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	// Human summary
	fmt.Println("\n" + strings.Repeat("─", 50))
	fmt.Println("✅ Pipeline complete!")
	fmt.Printf("   Steps executed: %d\n", len(results))
	valid, fixed := 0, 0
	for _, r := range results {
		if r.Valid {
			valid++
		}
		if r.Fixed {
			fixed++
		}
	}
	fmt.Printf("   Valid outputs:  %d/%d\n", valid, len(results))
	fmt.Printf("   Fixed outputs:  %d\n", fixed)
	if len(writtenFiles) > 0 {
		fmt.Printf("   Files written:  %d\n", len(writtenFiles))
	}
	if len(cfg.ValidationCommands) > 0 {
		if validationOK {
			fmt.Println("   Validation:     passed")
		} else {
			fmt.Println("   Validation:     failed")
		}
	}
	fmt.Println(strings.Repeat("─", 50))

	return nil
}

func tasksToSteps(tasks []models.Task) []string {
	var steps []string
	for _, t := range tasks {
		if strings.TrimSpace(t.Description) != "" {
			steps = append(steps, strings.TrimSpace(t.Description))
			continue
		}
		if strings.TrimSpace(t.Name) != "" {
			steps = append(steps, strings.TrimSpace(t.Name))
		}
	}
	return steps
}

func wrapContentAsFile(output string, taskDesc string) string {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return ""
	}
	l := strings.ToLower(taskDesc)
	path := ""
	switch {
	case strings.Contains(l, "index.html"):
		path = "index.html"
	case strings.Contains(l, "styles.css") || strings.Contains(l, "style.css"):
		path = "styles.css"
	case strings.Contains(l, ".html"):
		path = "index.html"
	case strings.Contains(l, ".css"):
		path = "styles.css"
	}
	if path == "" {
		return ""
	}
	return "// FILE: " + path + "\n" + output + "\n// END FILE"
}
