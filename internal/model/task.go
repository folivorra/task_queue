package model

import (
	"fmt"

	"github.com/folivorra/task_queue/pkg/apperrors"
)

type TaskStatus string

var (
	StatusQueued  TaskStatus = "queued"
	StatusRunning TaskStatus = "running"
	StatusDone    TaskStatus = "done"
	StatusFailed  TaskStatus = "failed"
)

type Task struct {
	ID         string     `json:"id"`
	Payload    string     `json:"payload"`
	MaxRetries int        `json:"max_retries"`
	Attempts   int        `json:"attempts"`
	Status     TaskStatus `json:"status"`
}

func ValidateTask(t Task) error {
	if t.ID == "" {
		return fmt.Errorf("%w: id is required", apperrors.ErrInvalidData)
	}
	if t.MaxRetries < 0 {
		return fmt.Errorf("%w: max_retries must be >= 0", apperrors.ErrInvalidData)
	}
	return nil
}
