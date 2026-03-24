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
	plan := planner.New(client)
	exec := executor.New(client, skillText, cfg.CommandAllow)
	fix := fixer.New(client, cfg.MaxFixRetries)
	ctxMgr := appctx.New(cfg.ContextBudget)

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

	if !jsonMode {
		fmt.Println("\n📋 Generated Plan:")
		for i, step := range originalPlan.Steps {
			fmt.Printf("   %d. %s\n", i+1, step)
		}
	}

	// ── Step 3: Edit Plan ──
	var editedPlan *models.Plan
	if flagNoEdit || jsonMode {
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
				for i, step := range editedPlan.Steps {
					fmt.Printf("   %d. %s\n", i+1, step)
				}
			}
		} else {
			editedPlan = originalPlan
		}
	}

	// ── Step 4: Diff & Learn ──
	d := diff.DiffPlans(originalPlan, editedPlan)
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
	validateFn := func(output string) (bool, string) {
		return validator.Validate(output)
	}
	fixFn := func(code string, reason string) (string, error) {
		return fix.Fix(code, reason)
	}
	commandFixFn := func(cmd string, output string, reason string) (string, error) {
		return fix.FixCommand(cmd, output, reason)
	}
	contextFn := func(currentCtx string, newOutput string) string {
		return ctxMgr.Update(newOutput)
	}

	results, execErr := exec.ExecuteAll(editedPlan, "", validateFn, fixFn, commandFixFn, contextFn)
	if execErr != nil && !jsonMode {
		fmt.Printf("\n❌ Execution error: %v\n", execErr)
	}

	// ── Step 6: Apply outputs to workspace ──
	var writtenFiles []string
	if app != nil {
		if !jsonMode {
			fmt.Println("\n📝 Applying outputs to workspace...")
		}
		for _, r := range results {
			files, applyErr := app.Apply(r.Output, r.StepIndex)
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

	// ── Step 7: Save to memory ──
	memStore.AddRecord(input, originalPlan.Steps, editedPlan.Steps, compiled.Keywords)
	_ = memStore.Save()

	// ── Output ──
	if jsonMode {
		// Machine-readable JSON output for Cline/Roo Code
		output := map[string]interface{}{
			"input":         input,
			"config": map[string]string{
				"provider": cfg.Provider,
				"model":    cfg.Model,
				"project":  cfg.ProjectPath,
			},
			"original_plan": originalPlan.Steps,
			"final_plan":    editedPlan.Steps,
			"results":       results,
			"files_written": writtenFiles,
			"plan_changed":  d.Changed,
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
	fmt.Println(strings.Repeat("─", 50))

	return nil
}
