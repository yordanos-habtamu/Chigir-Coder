package models

// Plan represents a structured execution plan with ordered, atomic steps.
type Plan struct {
	Steps []string `json:"steps"`
}

// Record stores input, plans, and tags for memory/history.
type Record struct {
	Input        string   `json:"input"`
	OriginalPlan []string `json:"original_plan"`
	RevisedPlan  []string `json:"revised_plan"`
	Tags         []string `json:"tags"`
	Timestamp    string   `json:"timestamp"`
}

// Learning captures patterns extracted from plan diffs.
type Learning struct {
	Pattern string   `json:"pattern"` // keyword, e.g. "auth"
	Added   []string `json:"added"`   // steps the user added
	Removed []string `json:"removed"` // steps the user removed
	Count   int      `json:"count"`   // how many times this pattern was seen
}

// StepResult holds the output of executing a single plan step.
type StepResult struct {
	StepIndex int    `json:"step_index"`
	Step      string `json:"step"`
	Output    string `json:"output"`
	Valid     bool   `json:"valid"`
	Fixed     bool   `json:"fixed"`
}

// ExecutionResult holds the full result of an execution run.
type ExecutionResult struct {
	Input   string       `json:"input"`
	Plan    Plan         `json:"plan"`
	Results []StepResult `json:"results"`
}
