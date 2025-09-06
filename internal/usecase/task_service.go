package usecase

import (
	"github.com/folivorra/task_queue/internal/model"
)

type TaskRepo interface {
	Save(task *model.Task) error
	Get(id string) (model.Task, error)
	UpdateStatus(id string, status model.TaskStatus) error
	List() []model.Task
}

type TaskService struct {
	repo  TaskRepo
	queue chan *model.Task
}

func NewTaskService(repo TaskRepo, queue chan *model.Task) *TaskService {
	return &TaskService{
		repo:  repo,
		queue: queue,
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

func (ts *TaskService) Get(id string) (model.Task, error) {
	return ts.repo.Get(id)
}

func (ts *TaskService) UpdateStatus(id string, status model.TaskStatus) error {
	return ts.repo.UpdateStatus(id, status)
}

func (ts *TaskService) List() []model.Task {
	return ts.repo.List()
}

func (ts *TaskService) PushToQueue(task *model.Task) {
	ts.queue <- task
}
