package api

import (
	"encoding/json"
	"net/http"
	"github.com/samaymehar/nolan/control-plane/internal/config"
	"github.com/samaymehar/nolan/control-plane/internal/db"
	"github.com/samaymehar/nolan/control-plane/internal/queue"
	"github.com/samaymehar/nolan/control-plane/internal/storage"
)

type Handler struct {
	storage  *storage.Storage
	producer *queue.Producer
	repo     *db.Repository
	config   *config.Config
}

func NewHandler(s *storage.Storage, p *queue.Producer, r *db.Repository, c *config.Config) *Handler {
	return &Handler{
		storage:  s,
		producer: p,
		repo:     r,
		config:   c,
	}
}

func RegisterRoutes(mux *http.ServeMux, h *Handler) {
	mux.HandleFunc("POST /api/v1/jobs", h.CreateJob)
	mux.HandleFunc("GET /api/v1/jobs/{job_id}", h.GetJob)
	mux.HandleFunc("GET /api/v1/jobs", h.ListJobs)
	mux.HandleFunc("POST /api/v1/upload-url", h.GetUploadURL)
	mux.HandleFunc("GET /api/v1/download-url/{video_id}", h.GetDownloadURL)
	mux.HandleFunc("GET /healthz", h.HealthCheck)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, map[string]string{
		"error": message,
		"code":  code,
	})
}
