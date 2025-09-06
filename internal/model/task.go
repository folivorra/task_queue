package model

import "fmt"

type Task struct {
	ID         string `json:"id"`
	Payload    string `json:"payload"`
	MaxRetries int    `json:"max_retries"`
}

func ValidateTask(t Task) error {
	if t.ID == "" {
		return fmt.Errorf("id is required")
	}
	if t.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be >= 0")
	}
	return nil
}
