package inmemory

import (
	"fmt"
	"sync"

	"github.com/folivorra/task_queue/internal/model"
	"github.com/folivorra/task_queue/pkg/apperrors"
)

type TaskInMemoryRepo struct {
	storage map[string]*model.Task
	sync.RWMutex
}

func NewTaskInMemoryRepo() *TaskInMemoryRepo {
	return &TaskInMemoryRepo{
		storage: make(map[string]*model.Task, 10),
	}
}

func (tr *TaskInMemoryRepo) Save(task *model.Task) error {
	tr.Lock()
	defer tr.Unlock()
	if _, ok := tr.storage[task.ID]; ok {
		return fmt.Errorf("%w: task already exist", apperrors.ErrAlreadyExists)
	}

	tr.storage[task.ID] = task

	return nil
}

func (tr *TaskInMemoryRepo) Get(id string) (model.Task, error) {
	tr.RLock()
	defer tr.RUnlock()
	taskPtr, ok := tr.storage[id]
	if !ok {
		return model.Task{}, fmt.Errorf("%w: task not found", apperrors.ErrNotFound)
	}

	return *taskPtr, nil
}

func (tr *TaskInMemoryRepo) UpdateStatus(id string, status model.TaskStatus) error {
	tr.Lock()
	defer tr.Unlock()

	task, ok := tr.storage[id]
	if !ok {
		return fmt.Errorf("%w: task not found", apperrors.ErrNotFound)
	}

	task.Status = status

	return nil
}

func (tr *TaskInMemoryRepo) List() []model.Task {
	tr.RLock()
	defer tr.RUnlock()

	tasks := make([]model.Task, 0, len(tr.storage))
	for _, t := range tr.storage {
		tasks = append(tasks, *t)
	}

	return tasks
}
