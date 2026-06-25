package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samaymehar/nolan/control-plane/internal/model"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateJob(ctx context.Context, job *model.Job) error {
	query := `
		INSERT INTO jobs (id, project_id, video_id, status, source_bucket, source_key, targets, attempt, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		job.ID, job.ProjectID, job.VideoID, job.Status, job.SourceBucket, job.SourceKey, job.Targets, job.Attempt, job.CreatedAt, job.UpdatedAt)
	return err
}

func (r *Repository) GetJob(ctx context.Context, jobID string) (*model.Job, error) {
	query := `
		SELECT id, project_id, video_id, status, source_bucket, source_key, targets, attempt, error_message, created_at, updated_at, completed_at
		FROM jobs
		WHERE id = $1
	`
	var job model.Job
	var errMsg *string
	err := r.pool.QueryRow(ctx, query, jobID).Scan(
		&job.ID, &job.ProjectID, &job.VideoID, &job.Status, &job.SourceBucket, &job.SourceKey, &job.Targets, &job.Attempt, &errMsg, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt)
	if err != nil {
		return nil, err
	}
	if errMsg != nil {
		job.ErrorMessage = *errMsg
	}
	return &job, nil
}

func (r *Repository) UpdateJobStatus(ctx context.Context, jobID, status string, attempt int, errorMsg string) error {
	query := `
		UPDATE jobs
		SET status = $1, attempt = $2, error_message = $3, updated_at = now()
		WHERE id = $4
	`
	_, err := r.pool.Exec(ctx, query, status, attempt, errorMsg, jobID)
	return err
}

func (r *Repository) ListJobs(ctx context.Context, limit, offset int, status string) ([]*model.Job, int, error) {
	query := `
		SELECT id, project_id, video_id, status, source_bucket, source_key, targets, attempt, error_message, created_at, updated_at, completed_at
		FROM jobs
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []*model.Job
	for rows.Next() {
		var job model.Job
		var errMsg *string
		err := rows.Scan(
			&job.ID, &job.ProjectID, &job.VideoID, &job.Status, &job.SourceBucket, &job.SourceKey, &job.Targets, &job.Attempt, &errMsg, &job.CreatedAt, &job.UpdatedAt, &job.CompletedAt)
		if err != nil {
			return nil, 0, err
		}
		if errMsg != nil {
			job.ErrorMessage = *errMsg
		}
		jobs = append(jobs, &job)
	}

	var count int
	countQuery := `SELECT count(*) FROM jobs WHERE ($1 = '' OR status = $1)`
	err = r.pool.QueryRow(ctx, countQuery, status).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	return jobs, count, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
