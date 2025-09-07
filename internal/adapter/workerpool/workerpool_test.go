package workerpool_test

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/folivorra/task_queue/internal/adapter/workerpool"
	"github.com/folivorra/task_queue/internal/model"
	"github.com/folivorra/task_queue/internal/repository/inmemory"
	"github.com/folivorra/task_queue/internal/usecase"
	"log/slog"
)

func TestWorkerPool_ProcessTasks(t *testing.T) {
	repo := inmemory.NewTaskInMemoryRepo()
	service := usecase.NewTaskService(repo)
	logger := slog.New(
		slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}),
	)
	taskQueue := make(chan *model.Task, 10)
	retryQueue := make(chan *model.Task, 10)

	wp := workerpool.NewWorkerPool(service, 2, taskQueue, retryQueue, &sync.WaitGroup{}, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wp.Run(ctx)

	task := &model.Task{ID: "task1", MaxRetries: 3}
	_ = service.Save(task)
	wp.PushToQueue(task)

	time.Sleep(600 * time.Millisecond)

	got, _ := service.Get("task1")
	if got.Status != model.StatusDone && got.Status != model.StatusFailed {
		t.Errorf("unexpected task status: %s", got.Status)
	}
}
