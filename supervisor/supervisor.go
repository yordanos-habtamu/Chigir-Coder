package supervisor

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yordanos-habtamu/PromptOs/models"
	"github.com/yordanos-habtamu/PromptOs/prompts"
)

// Decide returns a supervisor decision. If useAI is false, a safe default is used.
func Decide(client *models.Client, task models.Task, errReason string, attempts int, useAI bool) (models.SupervisorDecision, error) {
	if !useAI {
		return models.SupervisorDecision{
			Action: "abort",
			Reason: fmt.Sprintf("retries exceeded (%d). last error: %s", attempts, errReason),
		}, nil
	}
	if client == nil {
		return models.SupervisorDecision{
			Action: "abort",
			Reason: "AI supervisor requested but client is nil",
		}, nil
	}
	userPrompt := prompts.SupervisorPrompt(task, errReason, attempts)
	raw, err := client.Chat(prompts.SupervisorSystem, userPrompt)
	if err != nil {
		return models.SupervisorDecision{}, err
	}
	var decision models.SupervisorDecision
	if err := json.Unmarshal([]byte(raw), &decision); err != nil {
		return models.SupervisorDecision{}, fmt.Errorf("supervisor returned invalid JSON: %w", err)
	}
	decision.Action = strings.ToLower(strings.TrimSpace(decision.Action))
	if decision.Action == "" {
		decision.Action = "abort"
	}
	if decision.Reason == "" {
		decision.Reason = "no reason provided"
	}
	return decision, nil
}
