package main

import (
	"context"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/folivorra/task_queue/internal/adapter/rest"
	"github.com/folivorra/task_queue/internal/adapter/workerpool"
	"github.com/folivorra/task_queue/internal/model"
	"github.com/folivorra/task_queue/internal/repository/inmemory"
	"github.com/folivorra/task_queue/internal/usecase"
)

var (
	queueSize  int
	workersNum int
)

func main() {
	// ctx
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// random seed
	rand.Seed(time.Now().UnixNano())

	// logger
	logger := slog.New(
		slog.NewTextHandler(
			os.Stdout, &slog.HandlerOptions{
				Level:     slog.LevelDebug,
				AddSource: true,
			},
		),
	)

	// env
	getENV()
	logger.Debug("getting environment variables",
		slog.Int("queueSize", queueSize),
		slog.Int("workersNum", workersNum),
	)

	// repo service
	taskRepo := inmemory.NewTaskInMemoryRepo()
	taskService := usecase.NewTaskService(taskRepo)

	// worker pool
	wg := &sync.WaitGroup{}
	workerPool := workerpool.NewWorkerPool(taskService, workersNum,
		make(chan *model.Task, queueSize), make(chan *model.Task, queueSize), wg, logger)
	workerPool.Run(ctx)

	// controller
	taskController := rest.NewTaskController(taskService, workerPool)

	// mux
	mux := http.NewServeMux()
	mux.HandleFunc("/enqueue", taskController.Enqueue)
	mux.HandleFunc("/healthz", taskController.Healthcheck)
	mux.HandleFunc("/task", taskController.GetTask)
	mux.HandleFunc("/tasks", taskController.GetTaskList)

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

	// graceful shutdown
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
	wg.Wait()
	workerPool.Shutdown()
}

func getENV() {
	var err error

	queueSizeStr := os.Getenv("QUEUE_SIZE")
	if queueSizeStr == "" {
		queueSizeStr = "64"
	}
	queueSize, err = strconv.Atoi(queueSizeStr)
	if err != nil {
		queueSize = 64
	}

	workersNumStr := os.Getenv("WORKERS")
	if workersNumStr == "" {
		workersNumStr = "4"
	}
	workersNum, err = strconv.Atoi(workersNumStr)
	if err != nil {
		workersNum = 4
	}

}
