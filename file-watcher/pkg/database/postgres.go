package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	MaxPool  int
}

type DB struct {
	*sql.DB
}

// NewPostgresDB creates a new PostgreSQL connection pool
func NewPostgresDB(cfg Config) (*DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(cfg.MaxPool)
	db.SetMaxIdleConns(cfg.MaxPool / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{db}, nil
}

// Job represents a processing job
type Job struct {
	ID                   string
	FileHash             string
	Filename             string
	Status               string
	CreatedAt            time.Time
	ProcessingStartedAt  *time.Time
	CompletedAt          *time.Time
	PDFLocation          *string
	ErrorMessage         *string
	RetryCount           int
	MaxRetries           int
}

// CreateJob creates a new job in the database
// Returns the job ID if successful, or error if duplicate exists
func (db *DB) CreateJob(ctx context.Context, fileHash, filename string) (string, error) {
	var jobID string

	query := `
		INSERT INTO processing_jobs (file_hash, filename, status)
		VALUES ($1, $2, 'pending')
		ON CONFLICT (file_hash) DO NOTHING
		RETURNING id
	`

	err := db.QueryRowContext(ctx, query, fileHash, filename).Scan(&jobID)
	if err == sql.ErrNoRows {
		// Duplicate file, return nil to indicate skip
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to create job: %w", err)
	}

	return jobID, nil
}

// UpdateJobStatus updates the job status
func (db *DB) UpdateJobStatus(ctx context.Context, jobID, status string) error {
	query := `
		UPDATE processing_jobs
		SET status = $1,
		    processing_started_at = CASE WHEN $1 = 'processing' AND processing_started_at IS NULL THEN NOW() ELSE processing_started_at END,
		    completed_at = CASE WHEN $1 = 'completed' AND completed_at IS NULL THEN NOW() ELSE completed_at END
		WHERE id = $2::uuid
	`

	_, err := db.ExecContext(ctx, query, status, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

// GetJobByFileHash retrieves a job by file hash
func (db *DB) GetJobByFileHash(ctx context.Context, fileHash string) (*Job, error) {
	job := &Job{}

	query := `
		SELECT id, file_hash, filename, status, created_at,
		       processing_started_at, completed_at, pdf_location,
		       error_message, retry_count, max_retries
		FROM processing_jobs
		WHERE file_hash = $1
	`

	err := db.QueryRowContext(ctx, query, fileHash).Scan(
		&job.ID,
		&job.FileHash,
		&job.Filename,
		&job.Status,
		&job.CreatedAt,
		&job.ProcessingStartedAt,
		&job.CompletedAt,
		&job.PDFLocation,
		&job.ErrorMessage,
		&job.RetryCount,
		&job.MaxRetries,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}

// FindStuckJobs finds jobs that are stuck in processing (> 1 hour)
func (db *DB) FindStuckJobs(ctx context.Context) ([]*Job, error) {
	query := `
		SELECT id, file_hash, filename, status, created_at,
		       processing_started_at, retry_count, max_retries
		FROM processing_jobs
		WHERE status = 'processing'
		  AND processing_started_at < NOW() - INTERVAL '1 hour'
		  AND retry_count < max_retries
		ORDER BY processing_started_at ASC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to find stuck jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{}
		err := rows.Scan(
			&job.ID,
			&job.FileHash,
			&job.Filename,
			&job.Status,
			&job.CreatedAt,
			&job.ProcessingStartedAt,
			&job.RetryCount,
			&job.MaxRetries,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

// MarkJobForRetry marks a job for retry
func (db *DB) MarkJobForRetry(ctx context.Context, jobID string) (bool, error) {
	query := `SELECT mark_job_for_retry($1)`

	var success bool
	err := db.QueryRowContext(ctx, query, jobID).Scan(&success)
	if err != nil {
		return false, fmt.Errorf("failed to mark job for retry: %w", err)
	}

	return success, nil
}

// UpdateJobWithError updates a job with error message
func (db *DB) UpdateJobWithError(ctx context.Context, jobID, errorMessage string) error {
	query := `
		UPDATE processing_jobs
		SET status = 'failed',
		    error_message = $1
		WHERE id = $2
	`

	_, err := db.ExecContext(ctx, query, errorMessage, jobID)
	if err != nil {
		return fmt.Errorf("failed to update job with error: %w", err)
	}

	return nil
}
