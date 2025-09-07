package inmemory_test

import (
	"testing"

	"github.com/folivorra/task_queue/internal/model"
	"github.com/folivorra/task_queue/internal/repository/inmemory"
)

func TestTaskRepo_SaveAndGet(t *testing.T) {
	repo := inmemory.NewTaskInMemoryRepo()
	task := &model.Task{
		ID:         "task1",
		Payload:    "data",
		MaxRetries: 3,
	}

	if err := repo.Save(task); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	got, err := repo.Get("task1")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}

	if got.ID != task.ID || got.Payload != task.Payload {
		t.Errorf("got %+v, want %+v", got, task)
	}
}

func TestTaskRepo_UpdateStatus(t *testing.T) {
	repo := inmemory.NewTaskInMemoryRepo()
	task := &model.Task{ID: "t1", MaxRetries: 3}
	_ = repo.Save(task)

	if err := repo.UpdateStatus("t1", model.StatusRunning); err != nil {
		t.Fatalf("update status failed: %v", err)
	}

	got, _ := repo.Get("t1")
	if got.Status != model.StatusRunning {
		t.Errorf("status = %s, want %s", got.Status, model.StatusRunning)
	}
}
