package usecase_test

import (
	"context"
	"testing"

	"github.com/folivorra/task_queue/internal/model"
	"github.com/folivorra/task_queue/internal/repository/inmemory"
	"github.com/folivorra/task_queue/internal/usecase"
)

func TestTaskService_HandleTask(t *testing.T) {
	repo := inmemory.NewTaskInMemoryRepo()
	service := usecase.NewTaskService(repo)
	task := &model.Task{ID: "t1", MaxRetries: 3}

	if err := service.Save(task); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	ctx := context.Background()
	err := service.HandleTask(ctx, task)
	if err != nil && task.Status != model.StatusFailed {
		t.Errorf("expected failed task to be marked failed, got status %s", task.Status)
	}
}
