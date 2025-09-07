package workerpool

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/folivorra/task_queue/internal/model"
	"github.com/folivorra/task_queue/internal/usecase"
)

type WorkerPool struct {
	service    *usecase.TaskService
	workersNum int
	taskQueue  chan *model.Task
	retryQueue chan *model.Task
	wg         *sync.WaitGroup
	logger     *slog.Logger
}

func NewWorkerPool(service *usecase.TaskService, workersNum int, taskQueue chan *model.Task, retryQueue chan *model.Task, wg *sync.WaitGroup, logger *slog.Logger) *WorkerPool {
	return &WorkerPool{
		service:    service,
		workersNum: workersNum,
		taskQueue:  taskQueue,
		retryQueue: retryQueue,
		wg:         wg,
		logger:     logger,
	}
}

func (wp *WorkerPool) Run(ctx context.Context) {
	wp.wg.Add(1)
	go func() {
		defer wp.wg.Done()
		wp.retryCheck(ctx)
	}()

	for i := 0; i < wp.workersNum; i++ {
		wp.wg.Add(1)
		go func() {
			defer wp.wg.Done()
			wp.worker(ctx, i+1, wp.taskQueue)
		}()
	}
}

func (wp *WorkerPool) PushToQueue(task *model.Task) {
	wp.taskQueue <- task
}

func (wp *WorkerPool) worker(ctx context.Context, workerID int, queue <-chan *model.Task) {
	for {
		select {
		case <-ctx.Done():
			wp.logger.Info("worker context done",
				slog.Int("worker_id", workerID),
			)
			return
		case task, ok := <-queue:
			if !ok {
				continue
			}

			if err := wp.service.HandleTask(ctx, task); err != nil {
				wp.logger.Warn("failed to handle task",
					slog.Int("worker_id", workerID),
					slog.String("task_id", task.ID),
					slog.String("error", err.Error()),
				)

				if task.MaxRetries > task.Attempts {
					wp.retryQueue <- task
				} else {
					wp.logger.Warn("task failed due to max retries",
						slog.Int("worker_id", workerID),
						slog.String("task_id", task.ID),
					)
				}
			} else {
				wp.logger.Info("task successfully done",
					slog.Int("worker_id", workerID),
					slog.String("task_id", task.ID),
				)
			}
		}
	}
}

func (wp *WorkerPool) retryCheck(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			wp.logger.Info("retry worker context done")
			return
		case task, ok := <-wp.retryQueue:
			if !ok {
				continue
			}

			go func(task *model.Task) {
				backoff := calculateBackoff(task.Attempts)

				select {
				case <-ctx.Done():
					wp.logger.Info("retry worker context done")
					return
				case <-time.After(backoff):
					wp.taskQueue <- task
				}
			}(task)
		}
	}
}

func calculateBackoff(attempts int) time.Duration {
	baseDelay := 100 * time.Millisecond
	maxDelay := 5 * time.Second

	if attempts <= 1 {
		return baseDelay
	}

	delay := baseDelay * (1 << (attempts - 1))
	if delay > maxDelay {
		delay = maxDelay
	}

	jitter := rand.Int63n(int64(delay / 2))
	return delay + time.Duration(jitter)
}

func (wp *WorkerPool) Shutdown() {
	close(wp.taskQueue)
	close(wp.retryQueue)
}
