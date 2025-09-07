package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/folivorra/task_queue/internal/model"
)

type TaskRepo interface {
	Save(task *model.Task) error
	Get(id string) (*model.Task, error)
	UpdateStatus(id string, status model.TaskStatus) error
	List() []*model.Task
	IncAttempts(id string) error
}

type TaskService struct {
	repo TaskRepo
}

func NewTaskService(repo TaskRepo) *TaskService {
	return &TaskService{
		repo: repo,
	}
}

func (ts *TaskService) Save(task *model.Task) error {
	if err := model.ValidateTask(*task); err != nil {
		return err
	}

	task.Status = model.StatusQueued

	if err := ts.repo.Save(task); err != nil {
		return err
	}

	return nil
}

func (ts *TaskService) Get(id string) (*model.Task, error) {
	return ts.repo.Get(id)
}

func (ts *TaskService) UpdateStatus(id string, status model.TaskStatus) error {
	return ts.repo.UpdateStatus(id, status)
}

func (ts *TaskService) IncAttempts(id string) error {
	return ts.repo.IncAttempts(id)
}

func (ts *TaskService) List() []*model.Task {
	return ts.repo.List()
}

func (ts *TaskService) HandleTask(ctx context.Context, task *model.Task) error {
	if err := ts.UpdateStatus(task.ID, model.StatusRunning); err != nil {
		return err
	}

	if err := ts.IncAttempts(task.ID); err != nil {
		return err
	}

	if err := processing(ctx); err != nil {
		if updateErr := ts.UpdateStatus(task.ID, model.StatusFailed); updateErr != nil {
			return updateErr
		}
		return err
	}

	if err := ts.UpdateStatus(task.ID, model.StatusDone); err != nil {
		return err
	}

	return nil
}

func processing(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 + time.Duration(rand.Intn(4)*100)*time.Millisecond):
	}

	if rand.Intn(100) < 20 {
		return fmt.Errorf("simulated processing failed")
	}

	return nil
}
