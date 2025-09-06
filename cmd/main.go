package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/folivorra/task_queue/internal/adapter/rest"
	"github.com/folivorra/task_queue/internal/model"
	"github.com/folivorra/task_queue/internal/repository/inmemory"
	"github.com/folivorra/task_queue/internal/usecase"
)

func main() {
	// ctx
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	// logger
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
			},
		),
	)

	// queue_size get env
	queueSizeStr := os.Getenv("QUEUE_SIZE")
	if queueSizeStr == "" {
		queueSizeStr = "64"
	}
	queueSize, err := strconv.Atoi(queueSizeStr)
	if err != nil {
		logger.Warn("QUEUE_SIZE is invalid, set default",
			slog.Int("default_value", 64),
		)
		queueSize = 64
	}

	// queue
	taskQueue := make(chan *model.Task, queueSize)

	// repo service controller
	taskRepo := inmemory.NewTaskInMemoryRepo()
	taskService := usecase.NewTaskService(taskRepo, taskQueue)
	taskController := rest.NewTaskController(taskService)

	// mux
	mux := http.NewServeMux()
	mux.HandleFunc("/enqueue", taskController.Enqueue)
	mux.HandleFunc("/healthz", taskController.Healthcheck)

	// server
	server := rest.NewServer(&http.Server{
		Addr:    ":8080",
		Handler: mux,
	}, logger)

	go func() {
		if err := server.Run(); err != nil {
			logger.Error("server error",
				slog.String("err", err.Error()),
			)
		}
	}()

	// gracefull shutdown
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownCh
	logger.Info("received shutdown signal")

	if err := server.Stop(); err != nil {
		logger.Error("server stopped incorrectly",
			slog.String("err", err.Error()),
		)
	}

	cancel()
}
