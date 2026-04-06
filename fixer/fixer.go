package fixer

import (
	"fmt"

	"github.com/yordanos-habtamu/PromptOs/models"
	"github.com/yordanos-habtamu/PromptOs/prompts"
)

// Fixer retries failed outputs using correction prompts.
type Fixer struct {
	client     *models.Client
	maxRetries int
}

// New creates a Fixer with configurable retry count.
func New(client *models.Client, maxRetries int) *Fixer {
	if maxRetries <= 0 {
		maxRetries = 2
	}
	return &Fixer{client: client, maxRetries: maxRetries}
}

// Fix sends broken code + error details to the LLM for correction.
// Returns the fixed code or an error if all retries are exhausted.
func (f *Fixer) Fix(brokenCode string, errorDetail string, taskDescription string, requiresContent bool) (string, error) {
	userPrompt := prompts.FixerPrompt(brokenCode, errorDetail, taskDescription, requiresContent)
	fixed, err := f.client.Chat(prompts.FixerSystem, userPrompt)
	if err != nil {
		return "", fmt.Errorf("fixer LLM call failed: %w", err)
	}
	return fixed, nil
}

// FixCommand asks the LLM to repair a failed shell command.
func (f *Fixer) FixCommand(cmd string, output string, errorDetail string) (string, error) {
	userPrompt := prompts.CommandFixerPrompt(cmd, output, errorDetail)
	fixed, err := f.client.Chat(prompts.CommandFixerSystem, userPrompt)
	if err != nil {
		return "", fmt.Errorf("command fixer LLM call failed: %w", err)
	}
	return fixed, nil
}

// FixWithRetry attempts to fix the code, validating each attempt.
// validateFn should return (valid bool, reason string).
func (f *Fixer) FixWithRetry(brokenCode string, errorDetail string, taskDescription string, requiresContent bool, validateFn func(string) (bool, string)) (string, error) {
	code := brokenCode
	reason := errorDetail

	for attempt := 1; attempt <= f.maxRetries; attempt++ {
		fmt.Printf("    🔄 Fix attempt %d/%d\n", attempt, f.maxRetries)

		fixed, err := f.Fix(code, reason, taskDescription, requiresContent)
		if err != nil {
			return "", err
		}

		valid, newReason := validateFn(fixed)
		if valid {
			return fixed, nil
		}

		code = fixed
		reason = newReason
	}

	return code, fmt.Errorf("fixer exhausted %d retries, output may still be invalid", f.maxRetries)
}
