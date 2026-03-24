package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yordanos-habtamu/PromptOs/cmd"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "zen-coder",
		Short: "A deterministic AI coding orchestrator",
		Long: `zen-coder (PromptOs) is a deterministic execution pipeline for AI-assisted coding.

It takes messy user input, compiles it into structured prompts, generates a plan,
lets you revise it, learns from your edits, executes step-by-step, validates
outputs, and fixes errors automatically.`,
	}

	rootCmd.AddCommand(cmd.RunCmd)
	rootCmd.AddCommand(cmd.InitCmd)
	rootCmd.AddCommand(cmd.ServeCmd)
	rootCmd.AddCommand(cmd.SkillCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
