package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// InitCmd generates a default config file.
var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize zen-coder with a default config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configContent := `# zen-coder configuration
provider: openrouter
base_url: ""
model: ""
api_key: ""
project_path: "."
max_tokens: 300
context_budget: 800
output_mode: human
`
		path := "config.yaml"
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("⚠️  %s already exists. Overwrite? (y/n): ", path)
			var response string
			fmt.Scanln(&response)
			if response != "y" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		err := os.WriteFile(path, []byte(configContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		abs, _ := filepath.Abs(path)
		fmt.Printf("✅ Initialized default config at %s\n", abs)
		fmt.Println("Please edit config.yaml and add your API key.")
		return nil
	},
}
