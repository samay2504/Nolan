package model

import (
	"fmt"
	"strings"
	"time"
)

// Job status constants mirror the protobuf Status enum values.
const (
	StatusQueued     = "QUEUED"
	StatusProcessing = "PROCESSING"
	StatusCompleted  = "COMPLETED"
	StatusFailed     = "FAILED"
	StatusDLQ        = "DLQ"
)

// ValidStatuses enumerates all accepted status strings.
var ValidStatuses = map[string]bool{
	StatusQueued:     true,
	StatusProcessing: true,
	StatusCompleted:  true,
	StatusFailed:     true,
	StatusDLQ:        true,
}

// TargetRendition describes a single transcode target requested by the client.
type TargetRendition struct {
	Resolution string `json:"resolution"`
	Container  string `json:"container"`
}

// Job is the canonical domain model persisted in Postgres and cached in Valkey.
type Job struct {
	ID           string            `json:"id"`
	ProjectID    string            `json:"project_id"`
	VideoID      string            `json:"video_id"`
	Status       string            `json:"status"`
	SourceBucket string            `json:"source_bucket"`
	SourceKey    string            `json:"source_key"`
	Targets      []TargetRendition `json:"targets"`
	Attempt      int               `json:"attempt"`
	ErrorMessage string            `json:"error_message,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
}

// ---------- API request / response types ----------

// CreateJobRequest is the expected JSON body for POST /api/v1/jobs.
type CreateJobRequest struct {
	ProjectID    string            `json:"project_id"`
	VideoID      string            `json:"video_id,omitempty"`
	SourceBucket string            `json:"source_bucket"`
	SourceKey    string            `json:"source_key"`
	Targets      []TargetRendition `json:"targets"`
}

// Validate checks required fields and returns a user-facing error message.
func (r *CreateJobRequest) Validate() error {
	if strings.TrimSpace(r.ProjectID) == "" {
		return fmt.Errorf("project_id is required")
	}
	if strings.TrimSpace(r.SourceBucket) == "" {
		return fmt.Errorf("source_bucket is required")
	}
	if strings.TrimSpace(r.SourceKey) == "" {
		return fmt.Errorf("source_key is required")
	}
	if len(r.Targets) == 0 {
		return fmt.Errorf("at least one rendition target is required")
	}
	validResolutions := map[string]bool{
		"480P": true, "720P": true, "1080P": true, "4K": true,
	}
	for i, t := range r.Targets {
		if !validResolutions[strings.ToUpper(t.Resolution)] {
			return fmt.Errorf("targets[%d].resolution must be one of 480P, 720P, 1080P, 4K", i)
		}
		if strings.TrimSpace(t.Container) == "" {
			return fmt.Errorf("targets[%d].container is required", i)
		}
	}
	return nil
}

// JobResponse is the JSON envelope returned from job endpoints.
type JobResponse struct {
	Job       *Job   `json:"job"`
	UploadURL string `json:"upload_url,omitempty"`
}

// ListJobsResponse is the JSON envelope for paginated job lists.
type ListJobsResponse struct {
	Jobs   []*Job `json:"jobs"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

// UploadURLRequest is the JSON body for POST /api/v1/upload-url.
type UploadURLRequest struct {
	VideoID   string `json:"video_id"`
	Extension string `json:"extension"`
}

// Validate checks required fields.
func (r *UploadURLRequest) Validate() error {
	if strings.TrimSpace(r.VideoID) == "" {
		return fmt.Errorf("video_id is required")
	}
	if strings.TrimSpace(r.Extension) == "" {
		return fmt.Errorf("extension is required")
	}
	return nil
}

// UploadURLResponse wraps the presigned upload URL.
type UploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	VideoID   string `json:"video_id"`
	SourceKey string `json:"source_key"`
}

// DownloadURLRequest holds the parsed path parameter for download URL generation.
type DownloadURLRequest struct {
	VideoID string `json:"video_id"`
}

// DownloadURLResponse wraps the presigned download URL.
type DownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
}
