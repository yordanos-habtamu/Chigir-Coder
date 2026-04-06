package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/yordanos-habtamu/PromptOs/models"
)

// RunState is the persisted source of truth for a single execution.
type RunState struct {
	RunID     string                `json:"run_id"`
	Input     string                `json:"input"`
	StartedAt string                `json:"started_at"`
	Plan      models.Plan           `json:"plan"`
	State     models.ExecutionState `json:"state"`
}

// Store persists run state to a JSON file.
type Store struct {
	path string
	data RunState
}

func defaultStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "zen-coder", "state"), nil
}

// New creates a new state store with a fresh run id.
func New(input string, plan models.Plan) (*Store, error) {
	dir, err := defaultStateDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	runID := time.Now().Format("20060102-150405")
	path := filepath.Join(dir, fmt.Sprintf("run-%s.json", runID))
	rs := RunState{
		RunID:     runID,
		Input:     input,
		StartedAt: time.Now().Format(time.RFC3339),
		Plan:      plan,
		State: models.ExecutionState{
			CurrentTaskID:  0,
			CompletedTasks: []int{},
			FailedTasks:    []int{},
			GlobalContext:  map[string]string{},
		},
	}
	return &Store{path: path, data: rs}, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Data() RunState {
	return s.data
}

// Save writes the current state to disk.
func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *Store) SetCurrentTask(id int) {
	s.data.State.CurrentTaskID = id
}

func (s *Store) MarkCompleted(id int) {
	s.data.State.CompletedTasks = append(s.data.State.CompletedTasks, id)
}

func (s *Store) MarkFailed(id int) {
	s.data.State.FailedTasks = append(s.data.State.FailedTasks, id)
}

func (s *Store) UpdateTask(updated models.Task) {
	for i := range s.data.Plan.Tasks {
		if s.data.Plan.Tasks[i].ID == updated.ID {
			s.data.Plan.Tasks[i] = updated
			return
		}
	}
}
