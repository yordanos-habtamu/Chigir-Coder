package models

// Task represents a single unit of work in a plan.
type Task struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Status       string `json:"status"` // pending | running | done | failed
	Retries      int    `json:"retries"`
	MaxRetries   int    `json:"max_retries"`
	Dependencies []int  `json:"dependencies"`
	Output       string `json:"output,omitempty"`
	Error        string `json:"error,omitempty"`
}

// Plan represents a structured execution plan.
// Steps is kept for backward compatibility.
type Plan struct {
	Goal  string   `json:"goal,omitempty"`
	Tasks []Task   `json:"tasks,omitempty"`
	Steps []string `json:"steps,omitempty"`
}

// ExecutionState tracks current progress for a run.
type ExecutionState struct {
	CurrentTaskID  int               `json:"current_task_id"`
	CompletedTasks []int             `json:"completed_tasks"`
	FailedTasks    []int             `json:"failed_tasks"`
	GlobalContext  map[string]string `json:"global_context"`
}

// SupervisorDecision guides failure handling when retries are exhausted.
type SupervisorDecision struct {
	Action       string `json:"action"` // retry | modify | skip | abort
	Reason       string `json:"reason"`
	Modification string `json:"modification,omitempty"`
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
