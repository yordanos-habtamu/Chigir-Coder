package diff

import (
	"github.com/yordanos-habtamu/PromptOs/models"
	"github.com/yordanos-habtamu/PromptOs/utils"
)

// DiffResult holds the diff between original and revised plans.
type DiffResult struct {
	Added   []string // steps present in revised but not original
	Removed []string // steps present in original but not revised
	Changed bool     // whether any diff exists
}

// DiffPlans compares original and revised plans, returning added/removed steps.
func DiffPlans(original, revised *models.Plan) *DiffResult {
	origSet := toSet(original.Steps)
	revSet := toSet(revised.Steps)

	var added, removed []string

	for _, step := range revised.Steps {
		if !origSet[step] {
			added = append(added, step)
		}
	}

	for _, step := range original.Steps {
		if !revSet[step] {
			removed = append(removed, step)
		}
	}

	return &DiffResult{
		Added:   added,
		Removed: removed,
		Changed: len(added) > 0 || len(removed) > 0,
	}
}

// ExtractLearnings converts a diff into learning patterns, mapped to keywords
// from the user input.
func ExtractLearnings(d *DiffResult, userInput string) []models.Learning {
	if !d.Changed {
		return nil
	}

	keywords := utils.ExtractKeywords(userInput)
	var learnings []models.Learning

	for _, kw := range keywords {
		learnings = append(learnings, models.Learning{
			Pattern: kw,
			Added:   d.Added,
			Removed: d.Removed,
			Count:   1,
		})
	}
	return learnings
}

func toSet(items []string) map[string]bool {
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
