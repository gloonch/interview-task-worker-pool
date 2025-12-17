package router

import (
	"interview-task-worker-pool/internal/http/handlers"
	"net/http"
)

func New(handler *handlers.TaskHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /tasks", handler.Create)
	mux.HandleFunc("GET /tasks", handler.List)
	mux.HandleFunc("GET /tasks/{id}", handler.Get)

	return mux
}
