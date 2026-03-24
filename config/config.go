package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)


// Provider presets map provider names to their default base URLs.
var ProviderPresets = map[string]string{
	"openrouter": "https://openrouter.ai/api/v1",
	"openai":     "https://api.openai.com/v1",
	"anthropic":  "https://api.anthropic.com/v1",
	"nvidia":     "https://integrate.api.nvidia.com/v1",
	"ollama":     "http://localhost:11434/v1",
	"custom":     "",
}

// Default models per provider (user can override).
var DefaultModels = map[string]string{
	"openrouter": "arcee-ai/trinity-large-preview:free",
	"openai":     "gpt-4o-mini",
	"anthropic":  "claude-3-5-haiku-20241022",
	// NVIDIA NIM/OpenAI-compatible models vary by account/region; set explicitly if needed.
	"nvidia":     "qwen/qwen3.5-122b-a10b",
	"ollama":     "qwen2.5-coder:7b",
	"custom":     "gpt-4o-mini",
}

// Config holds all runtime configuration.
type Config struct {
	Provider       string `mapstructure:"provider"`
	BaseURL        string `mapstructure:"base_url"`
	APIKey         string `mapstructure:"api_key"`
	Model          string `mapstructure:"model"`
	Skill          string `mapstructure:"skill"`
	ProjectPath    string `mapstructure:"project_path"`
	MaxTokens      int    `mapstructure:"max_tokens"`
	MaxFixRetries  int    `mapstructure:"max_fix_retries"`
	ContextBudget  int    `mapstructure:"context_budget"`
	CommandAllow  []string `mapstructure:"command_allowlist"`
	OutputMode     string `mapstructure:"output_mode"` // "human" or "json"
	MCPPort        int    `mapstructure:"mcp_port"`    // 0 = stdio transport
	ConfigFile     string `mapstructure:"config_file"`
}

// Load reads config from file, env vars, and applies provider presets.
// Priority: CLI flags (applied after) > env vars > config file > defaults
func Load() (*Config, error) {
	v := viper.New()

	// ── Defaults ──
	v.SetDefault("provider", "openrouter")
	v.SetDefault("base_url", "")
	v.SetDefault("model", "")
	v.SetDefault("api_key", "")
	v.SetDefault("skill", "auto")
	v.SetDefault("project_path", ".")
	v.SetDefault("max_tokens", 300)
	v.SetDefault("max_fix_retries", 1)
	v.SetDefault("context_budget", 800)
	v.SetDefault("command_allowlist", []string{"ls", "pwd", "mkdir", "touch", "cat", "echo", "cp", "mv", "git", "go", "npm", "npx", "node", "python", "python3", "pip", "pip3"})
	v.SetDefault("output_mode", "human")
	v.SetDefault("mcp_port", 0)

	// ── Config file search ──
	// 1. Current directory
	// 2. Home directory ~/.config/zen-coder/
	// 3. /etc/zen-coder/
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("config")
	if home, err := os.UserHomeDir(); err == nil {
		v.AddConfigPath(filepath.Join(home, ".config", "zen-coder"))
	}
	v.AddConfigPath("/etc/zen-coder")

	// ── Environment variables ──
	v.SetEnvPrefix("ZEN")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// Read config file (optional — it's okay if not found)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config file error: %w", err)
		}
		// No config file found — that's fine, use env + defaults
	} else {
		v.Set("config_file", v.ConfigFileUsed())
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("config parse error: %w", err)
	}

	// ── Apply provider preset for base_url if empty ──
	if c.BaseURL == "" {
		if preset, ok := ProviderPresets[c.Provider]; ok {
			c.BaseURL = preset
		} else {
			return nil, fmt.Errorf("unknown provider %q — use: openrouter, openai, anthropic, nvidia, ollama, custom", c.Provider)
		}
	}

	// ── Apply default model per provider if empty ──
	if c.Model == "" {
		if defaultModel, ok := DefaultModels[c.Provider]; ok {
			c.Model = defaultModel
		}
	}

	// ── Resolve project path ──
	if c.ProjectPath == "" || c.ProjectPath == "." {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot detect working directory: %w", err)
		}
		c.ProjectPath = cwd
	} else {
		abs, err := filepath.Abs(c.ProjectPath)
		if err != nil {
			return nil, fmt.Errorf("invalid project path: %w", err)
		}
		c.ProjectPath = abs
	}

	return &c, nil
}

// ApplyOverrides merges CLI flag values into the config.
// Only non-zero/non-empty values override.
func (c *Config) ApplyOverrides(provider, baseURL, apiKey, model, skill, projectPath, outputMode string, maxTokens int) {
	if provider != "" {
		c.Provider = provider
		// Re-apply preset if base URL wasn't also overridden
		if baseURL == "" {
			if preset, ok := ProviderPresets[provider]; ok {
				c.BaseURL = preset
			}
		}
		// Re-apply default model if model wasn't also overridden
		if model == "" {
			if defaultModel, ok := DefaultModels[provider]; ok {
				c.Model = defaultModel
			}
		}
	}
	if baseURL != "" {
		c.BaseURL = baseURL
	}
	if apiKey != "" {
		c.APIKey = apiKey
	}
	if model != "" {
		c.Model = model
	}
	if skill != "" {
		c.Skill = skill
	}
	if projectPath != "" {
		if projectPath == "." {
			if cwd, err := os.Getwd(); err == nil {
				c.ProjectPath = cwd
			}
		} else if abs, err := filepath.Abs(projectPath); err == nil {
			c.ProjectPath = abs
		}
	}
	if outputMode != "" {
		c.OutputMode = outputMode
	}
	if maxTokens > 0 {
		c.MaxTokens = maxTokens
	}
}

// Validate checks that required fields are set.
func (c *Config) Validate() error {
	if c.APIKey == "" && c.Provider != "ollama" {
		return fmt.Errorf("API key is required. Set via --api-key, ZEN_API_KEY env var, or config.yaml")
	}
	if c.BaseURL == "" {
		return fmt.Errorf("base URL is required. Set provider or use --base-url")
	}
	if c.Model == "" {
		return fmt.Errorf("model is required. Set via --model, ZEN_MODEL env var, or config.yaml")
	}
	return nil
}

// Summary returns a human-readable config summary (masks the API key).
func (c *Config) Summary() string {
	maskedKey := "(not set)"
	if c.APIKey != "" {
		if len(c.APIKey) > 8 {
			maskedKey = c.APIKey[:4] + "..." + c.APIKey[len(c.APIKey)-4:]
		} else {
			maskedKey = "****"
		}
	}
	configFile := c.ConfigFile
	if configFile == "" {
		configFile = "(none)"
	}
	return fmt.Sprintf(`  Provider:    %s
  Base URL:    %s
  Model:       %s
  API Key:     %s
  Skill:       %s
  Project:     %s
  Max Tokens:  %d
  Cmd Allow:   %v
  Output Mode: %s
  Config File: %s`, c.Provider, c.BaseURL, c.Model, maskedKey, c.Skill, c.ProjectPath, c.MaxTokens, c.CommandAllow, c.OutputMode, configFile)
}

// ListProviders returns all supported provider names.
func ListProviders() []string {
	return []string{"openrouter", "openai", "anthropic", "nvidia", "ollama", "custom"}
}
