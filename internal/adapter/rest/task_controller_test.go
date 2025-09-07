package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/folivorra/task_queue/internal/adapter/rest"
	"github.com/folivorra/task_queue/internal/adapter/workerpool"
	"github.com/folivorra/task_queue/internal/model"
	"github.com/folivorra/task_queue/internal/repository/inmemory"
	"github.com/folivorra/task_queue/internal/usecase"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func setupTestServer(t *testing.T) (*httptest.Server, *usecase.TaskService, *workerpool.WorkerPool, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	logger := slog.New(
		slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}),
	)

	taskRepo := inmemory.NewTaskInMemoryRepo()
	taskService := usecase.NewTaskService(taskRepo)

	taskQueue := make(chan *model.Task, 10)
	retryQueue := make(chan *model.Task, 10)
	wg := &sync.WaitGroup{}
	wp := workerpool.NewWorkerPool(taskService, 2, taskQueue, retryQueue, wg, logger)
	wp.Run(ctx)

	taskController := rest.NewTaskController(taskService, wp)

	mux := http.NewServeMux()
	mux.HandleFunc("/enqueue", taskController.Enqueue)
	mux.HandleFunc("/tasks", taskController.GetTaskList)
	mux.HandleFunc("/healthz", taskController.Healthcheck)

	server := httptest.NewServer(mux)

	return server, taskService, wp, cancel
}

func TestEnqueueEndpoint(t *testing.T) {
	server, _, wp, cancel := setupTestServer(t)
	defer server.Close()
	defer cancel()

	task := model.CreateTaskRequest{
		ID:         "test1",
		Payload:    "payload1",
		MaxRetries: 3,
	}
	body, _ := json.Marshal(task)

	resp, err := http.Post(server.URL+"/enqueue", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var created model.Task
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if created.ID != task.ID {
		t.Errorf("expected ID %s, got %s", task.ID, created.ID)
	}

	wp.Shutdown()
}

func TestGetTasksEndpoint(t *testing.T) {
	server, taskService, wp, cancel := setupTestServer(t)
	defer server.Close()
	defer cancel()

	// Создаём несколько задач
	tasks := []*model.Task{
		{ID: "task1", Payload: "p1", MaxRetries: 3},
		{ID: "task2", Payload: "p2", MaxRetries: 2},
	}
	for _, task := range tasks {
		if err := taskService.Save(task); err != nil {
			t.Fatalf("failed to save task: %v", err)
		}
	}

	resp, err := http.Get(server.URL + "/tasks")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var got []*model.Task
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(got) != len(tasks) {
		t.Errorf("expected %d tasks, got %d", len(tasks), len(got))
	}

	wp.Shutdown()
}

func TestHealthcheckEndpoint(t *testing.T) {
	server, _, _, cancel := setupTestServer(t)
	defer server.Close()
	defer cancel()

	resp, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var data map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if data["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", data["status"])
	}
}
