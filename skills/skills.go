package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func DetectSkill(task string) string {
	t := strings.ToLower(task)
	switch {
	case containsAny(t, "landing page", "landing-page", "marketing", "hero", "soccer", "football", "club", "brand"):
		return "landing-page"
	case containsAny(t, "portfolio", "case study", "personal site"):
		return "portfolio"
	case containsAny(t, "dashboard", "admin", "analytics"):
		return "dashboard"
	case containsAny(t, "docs", "documentation", "readme", "guide"):
		return "docs"
	case containsAny(t, "api", "rest", "graphql", "endpoint", "server"):
		return "api"
	case containsAny(t, "cli", "command line", "terminal tool"):
		return "cli"
	case containsAny(t, "refactor", "cleanup", "restructure"):
		return "refactor"
	case containsAny(t, "test", "testing", "unit test", "integration test"):
		return "tests"
	case containsAny(t, "pipeline", "etl", "data flow"):
		return "data-pipeline"
	case containsAny(t, "next", "nextjs", "next.js"):
		return "next"
	case containsAny(t, "react", "vite"):
		return "react"
	default:
		return "general"
	}
}

func LoadSkill(name string) (string, error) {
	n := strings.TrimSpace(strings.ToLower(name))
	if n == "" || n == "none" {
		return "", nil
	}
	fileName := fmt.Sprintf("%s.md", n)

	// Try current working directory first
	if data, err := os.ReadFile(filepath.Join("skills", fileName)); err == nil {
		return string(data), nil
	}

	// Try alongside the executable
	if dir, err := ExecutableSkillsDir(); err == nil {
		if data, err2 := os.ReadFile(filepath.Join(dir, fileName)); err2 == nil {
			return string(data), nil
		}
	}

	// Try user config dir
	if dir, err := UserSkillsDir(); err == nil {
		if data, err2 := os.ReadFile(filepath.Join(dir, fileName)); err2 == nil {
			return string(data), nil
		}
	}

	// Try system dir
	if data, err := os.ReadFile(filepath.Join(SystemSkillsDir(), fileName)); err == nil {
		return string(data), nil
	}

	return "", fmt.Errorf("skill file not found: %s", fileName)
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
