package handlers

import (
	"encoding/json"
	"errors"
	"interview-task-worker-pool/internal/domain"
	"interview-task-worker-pool/internal/http/dto"
	"interview-task-worker-pool/internal/service"
	"interview-task-worker-pool/internal/workerpool"
	"net/http"
	"strconv"
)

type TaskService interface {
	CreateTask(title, description string) (domain.Task, error)
	GetTask(id int64) (domain.Task, error)
	ListTasks() ([]domain.Task, error)
}

type TaskHandler struct {
	taskService TaskService
}

func New(taskService TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

// POST /tasks
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())

		return
	}

	task, err := h.taskService.CreateTask(req.Title, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, service.ErrInvalidInput.Error())
			return
		case errors.Is(err, workerpool.ErrPoolFull):
			writeJSON(w, http.StatusServiceUnavailable, dto.TaskResponse{
				ID:          task.ID,
				Title:       task.Title,
				Description: task.Description,
				Status:      string(task.Status),
				Error:       task.Error,
			})
			return
		case errors.Is(err, workerpool.ErrPoolClosed):
			writeJSON(w, http.StatusServiceUnavailable, dto.TaskResponse{
				ID:          task.ID,
				Title:       task.Title,
				Description: task.Description,
				Status:      string(task.Status),
				Error:       task.Error,
			})
			return
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}
	}

	response := dto.TaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		Error:       task.Error,
	}

	writeJSON(w, http.StatusCreated, response)
}

// GET /tasks/{id}
func (h *TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, service.ErrInvalidID.Error())

		return
	}

	task, err := h.taskService.GetTask(id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidID):
			writeError(w, http.StatusBadRequest, service.ErrInvalidID.Error())
			return
		case errors.Is(err, service.ErrNotFound):
			writeError(w, http.StatusNotFound, service.ErrNotFound.Error())
			return
		default:
			writeError(w, http.StatusInternalServerError, "failed getting task")
			return
		}
	}

	response := dto.TaskResponse{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		Error:       task.Error,
	}

	writeJSON(w, http.StatusOK, response)
}

// GET /tasks
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.taskService.ListTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed getting tasks")

		return
	}

	response := make([]dto.TaskSummaryResponse, 0, len(tasks))
	for _, task := range tasks {
		response = append(response, dto.TaskSummaryResponse{
			ID:     task.ID,
			Title:  task.Title,
			Status: string(task.Status),
		})
	}

	writeJSON(w, http.StatusOK, response)
}
