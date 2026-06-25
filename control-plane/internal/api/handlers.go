package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samaymehar/nolan/control-plane/internal/model"
)

func (h *Handler) CreateJob(w http.ResponseWriter, r *http.Request) {
	var req model.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload")
		return
	}

	jobID := ulid.Make().String()
	videoID := req.VideoID
	if videoID == "" {
		videoID = ulid.Make().String()
	}

	job := &model.Job{
		ID:           jobID,
		ProjectID:    req.ProjectID,
		VideoID:      videoID,
		Status:       model.StatusQueued,
		SourceBucket: "raw-input",
		SourceKey:    req.SourceKey,
		Targets:      req.Targets,
		Attempt:      0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.repo.CreateJob(r.Context(), job); err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to create job record")
		return
	}

	if err := h.producer.EnqueueJob(r.Context(), job); err != nil {
		// We could implement retry/rollback here, but keeping it simple.
		respondError(w, http.StatusInternalServerError, "QUEUE_ERROR", "Failed to enqueue job")
		return
	}

	respondJSON(w, http.StatusCreated, job)
}

func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID := r.PathValue("job_id")
	if jobID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "job_id is required")
		return
	}

	// Try valkey hot status first (omitted for brevity, could be added in model.Job retrieval from valkey)
	// Fallback to DB
	job, err := h.repo.GetJob(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Job not found")
		return
	}

	if hotData, err := h.producer.GetJobHotData(r.Context(), jobID); err == nil && len(hotData) > 0 {
		if st, ok := hotData["status"]; ok {
			job.Status = st
		}
		if errStr, ok := hotData["error"]; ok {
			job.ErrorMessage = errStr
		}
		if attStr, ok := hotData["attempt"]; ok {
			if att, err := strconv.Atoi(attStr); err == nil {
				job.Attempt = att
			}
		}
	}

	respondJSON(w, http.StatusOK, job)
}

func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	status := r.URL.Query().Get("status")

	limit := 10
	offset := 0
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}
	if offsetStr != "" {
		offset, _ = strconv.Atoi(offsetStr)
	}

	jobs, count, err := h.repo.ListJobs(r.Context(), limit, offset, status)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "DB_ERROR", "Failed to list jobs")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"total": count,
	})
}

func (h *Handler) GetUploadURL(w http.ResponseWriter, r *http.Request) {
	var req model.UploadURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON payload")
		return
	}

	if req.Extension == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "extension is required")
		return
	}

	videoID := ulid.Make().String()
	url, err := h.storage.PresignedUpload(r.Context(), videoID, req.Extension, h.config.UploadURLTTL)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to generate upload URL")
		return
	}

	respondJSON(w, http.StatusOK, model.UploadURLResponse{
		UploadURL: url,
		VideoID:   videoID,
		SourceKey: videoID + "/source" + req.Extension,
	})
}

func (h *Handler) GetDownloadURL(w http.ResponseWriter, r *http.Request) {
	videoID := r.PathValue("video_id")
	key := r.URL.Query().Get("key")
	if videoID == "" || key == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "video_id and key are required")
		return
	}

	url, err := h.storage.PresignedDownload(r.Context(), "processed", videoID+"/"+key, h.config.DownloadURLTTL)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "STORAGE_ERROR", "Failed to generate download URL")
		return
	}

	respondJSON(w, http.StatusOK, model.DownloadURLResponse{
		DownloadURL: url,
	})
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Ping(r.Context()); err != nil {
		respondError(w, http.StatusServiceUnavailable, "DB_UNAVAILABLE", "Database ping failed")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
