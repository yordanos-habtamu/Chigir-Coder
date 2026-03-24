package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func UserSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zen-coder", "skills"), nil
}

func SystemSkillsDir() string {
	return filepath.Join(string(os.PathSeparator), "etc", "zen-coder", "skills")
}

func LocalSkillsDir() string {
	return filepath.Join("skills")
}

func ExecutableSkillsDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), "skills"), nil
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func ListSkills() ([]string, error) {
	seen := make(map[string]struct{})
	paths := []string{}

	if p := LocalSkillsDir(); p != "" {
		paths = append(paths, p)
	}
	if p, err := ExecutableSkillsDir(); err == nil {
		paths = append(paths, p)
	}
	if p, err := UserSkillsDir(); err == nil {
		paths = append(paths, p)
	}
	paths = append(paths, SystemSkillsDir())

	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(name, ".md") {
				base := strings.TrimSuffix(name, ".md")
				seen[base] = struct{}{}
			}
		}
	}

	var out []string
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out, nil
}

func SkillTemplate(name string) string {
	return fmt.Sprintf(`SKILL: %s
GOAL: Describe the desired outcome.
STRUCTURE: Outline major sections or steps.
STYLE: Tone, aesthetics, conventions.
DELIVERABLES: Files or outputs to produce.
CONSTRAINTS: Any hard rules or limits.
`, name)
}
