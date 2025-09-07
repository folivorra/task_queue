package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/folivorra/task_queue/internal/adapter/workerpool"
	"github.com/folivorra/task_queue/internal/model"
	"github.com/folivorra/task_queue/internal/usecase"
	"github.com/folivorra/task_queue/pkg/apperrors"
)

type TaskController struct {
	service   *usecase.TaskService
	processor *workerpool.WorkerPool
}

func NewTaskController(service *usecase.TaskService, processor *workerpool.WorkerPool) *TaskController {
	return &TaskController{
		service:   service,
		processor: processor,
	}
}

func (tc *TaskController) Enqueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if r.Body == nil {
		writeJSONError(w, http.StatusBadRequest, "empty body")
		return
	}
	defer r.Body.Close()

	var req model.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	task := &model.Task{
		ID:         req.ID,
		Payload:    req.Payload,
		MaxRetries: req.MaxRetries,
	}

	if err := tc.service.Save(task); err != nil {
		switch {
		case errors.Is(err, apperrors.ErrAlreadyExists):
			writeJSONError(w, http.StatusConflict, err.Error())
		case errors.Is(err, apperrors.ErrInvalidData):
			writeJSONError(w, http.StatusBadRequest, err.Error())
		default:
			writeJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	tc.processor.PushToQueue(task)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}

func (tc *TaskController) Healthcheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (tc *TaskController) GetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ids := r.URL.Query().Get("id")
	if ids == "" {
		writeJSONError(w, http.StatusBadRequest, "missing id parameter")
		return
	}

	task, err := tc.service.Get(ids)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrNotFound):
			writeJSONError(w, http.StatusNotFound, err.Error())
		default:
			writeJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(task); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}

func (tc *TaskController) GetTaskList(w http.ResponseWriter, r *http.Request) {
	tasks := tc.service.List()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
	}
}
