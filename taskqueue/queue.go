package taskqueue

import "github.com/yordanos-habtamu/PromptOs/models"

// Queue provides deterministic task selection (FIFO with dependencies).
type Queue struct {
	tasks []models.Task
}

func New(tasks []models.Task) *Queue {
	return &Queue{tasks: tasks}
}

// Next returns the first pending task whose dependencies are satisfied.
func (q *Queue) Next(completed map[int]bool) *models.Task {
	for i := range q.tasks {
		t := q.tasks[i]
		if t.Status != "pending" {
			continue
		}
		if depsMet(t.Dependencies, completed) {
			return &q.tasks[i]
		}
	}
	return nil
}

func depsMet(deps []int, completed map[int]bool) bool {
	for _, id := range deps {
		if !completed[id] {
			return false
		}
	}
	return true
}
