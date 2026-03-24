package learning

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yordanos-habtamu/PromptOs/models"
)

func getLearningFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zen-coder", "memory", "learning.json")
}

// Store manages persistent learning patterns.
type Store struct {
	Patterns []models.Learning `json:"patterns"`
}

// Load reads learning patterns from disk.
func Load() (*Store, error) {
	s := &Store{}
	data, err := os.ReadFile(getLearningFile())
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil // fresh start
		}
		return nil, fmt.Errorf("failed to read learning store: %w", err)
	}
	if err := json.Unmarshal(data, &s.Patterns); err != nil {
		return nil, fmt.Errorf("failed to parse learning store: %w", err)
	}
	return s, nil
}

// Save writes learning patterns to disk.
func (s *Store) Save() error {
	path := getLearningFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.Patterns, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// AddLearnings merges new learnings into the store. If a pattern already
// exists, its added/removed steps are appended and the count incremented.
func (s *Store) AddLearnings(newLearnings []models.Learning) {
	for _, nl := range newLearnings {
		found := false
		for i, existing := range s.Patterns {
			if existing.Pattern == nl.Pattern {
				s.Patterns[i].Added = appendUnique(existing.Added, nl.Added...)
				s.Patterns[i].Removed = appendUnique(existing.Removed, nl.Removed...)
				s.Patterns[i].Count++
				found = true
				break
			}
		}
		if !found {
			s.Patterns = append(s.Patterns, nl)
		}
	}
}

// GetHints returns a formatted string of relevant learnings for the given keywords.
func (s *Store) GetHints(keywords []string) string {
	var hints []string
	for _, p := range s.Patterns {
		for _, kw := range keywords {
			if strings.EqualFold(p.Pattern, kw) && p.Count > 0 {
				if len(p.Added) > 0 {
					hints = append(hints, fmt.Sprintf("For '%s', users typically ADD: %s",
						p.Pattern, strings.Join(p.Added, "; ")))
				}
				if len(p.Removed) > 0 {
					hints = append(hints, fmt.Sprintf("For '%s', users typically REMOVE: %s",
						p.Pattern, strings.Join(p.Removed, "; ")))
				}
				break
			}
		}
	}
	return strings.Join(hints, "\n")
}

func appendUnique(existing []string, items ...string) []string {
	set := make(map[string]bool, len(existing))
	for _, e := range existing {
		set[e] = true
	}
	for _, item := range items {
		if !set[item] {
			existing = append(existing, item)
			set[item] = true
		}
	}
	return existing
}
