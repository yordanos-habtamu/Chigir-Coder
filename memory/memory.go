package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yordanos-habtamu/PromptOs/models"
)

func getHistoryFile() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "zen-coder", "memory", "history.json")
}

// Store handles persistent storage of execution records.
type Store struct {
	Records []models.Record `json:"records"`
}

// Load reads history from disk.
func Load() (*Store, error) {
	s := &Store{}
	data, err := os.ReadFile(getHistoryFile())
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, fmt.Errorf("failed to read history: %w", err)
	}
	if err := json.Unmarshal(data, &s.Records); err != nil {
		return nil, fmt.Errorf("failed to parse history: %w", err)
	}
	return s, nil
}

// Save writes history to disk.
func (s *Store) Save() error {
	path := getHistoryFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.Records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// AddRecord stores a new execution record with timestamp.
func (s *Store) AddRecord(input string, originalPlan, revisedPlan []string, tags []string) {
	s.Records = append(s.Records, models.Record{
		Input:        input,
		OriginalPlan: originalPlan,
		RevisedPlan:  revisedPlan,
		Tags:         tags,
		Timestamp:    time.Now().Format(time.RFC3339),
	})
}

// FindSimilar looks for past records that share keywords with the given tags.
// Returns the best matching record or nil.
func (s *Store) FindSimilar(tags []string) *models.Record {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}

	var best *models.Record
	bestScore := 0

	for i := range s.Records {
		score := 0
		for _, t := range s.Records[i].Tags {
			if tagSet[t] {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			best = &s.Records[i]
		}
	}

	if bestScore == 0 {
		return nil
	}
	return best
}
