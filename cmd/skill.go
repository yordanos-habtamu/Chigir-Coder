package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yordanos-habtamu/PromptOs/skills"
)

var (
	flagSkillFile  string
	flagSkillForce bool
)

// SkillCmd manages skill templates.
var SkillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage skill templates",
}

// skillAddCmd adds a skill template.
var skillAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a skill template to user config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.ToLower(strings.TrimSpace(args[0]))
		if name == "" {
			return fmt.Errorf("skill name is required")
		}
		dir, err := skills.UserSkillsDir()
		if err != nil {
			return err
		}
		if err := skills.EnsureDir(dir); err != nil {
			return err
		}
		target := filepath.Join(dir, name+".md")
		if !flagSkillForce {
			if _, err := os.Stat(target); err == nil {
				return fmt.Errorf("skill already exists: %s (use --force to overwrite)", target)
			}
		}

		var content []byte
		if flagSkillFile != "" {
			data, err := os.ReadFile(flagSkillFile)
			if err != nil {
				return fmt.Errorf("failed to read %s: %w", flagSkillFile, err)
			}
			content = data
		} else {
			content = []byte(skills.SkillTemplate(name))
		}

		if err := os.WriteFile(target, content, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", target, err)
		}
		fmt.Printf("✅ Added skill: %s\n", target)
		return nil
	},
}

// skillListCmd lists available skills.
var skillListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		items, err := skills.ListSkills()
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("(no skills found)")
			return nil
		}
		for _, s := range items {
			fmt.Println(s)
		}
		return nil
	},
}

func init() {
	SkillCmd.AddCommand(skillAddCmd)
	SkillCmd.AddCommand(skillListCmd)

	skillAddCmd.Flags().StringVar(&flagSkillFile, "file", "", "Path to a skill template file to import")
	skillAddCmd.Flags().BoolVar(&flagSkillForce, "force", false, "Overwrite if skill already exists")
}
