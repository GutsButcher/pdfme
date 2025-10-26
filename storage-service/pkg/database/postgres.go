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

// GetJobByID retrieves a job by ID
func (db *DB) GetJobByID(ctx context.Context, jobID string) (*Job, error) {
	job := &Job{}

	query := `
		SELECT id, file_hash, filename, status, created_at,
		       processing_started_at, completed_at, pdf_location,
		       error_message, retry_count, max_retries
		FROM processing_jobs
		WHERE id = $1
	`

	err := db.QueryRowContext(ctx, query, jobID).Scan(
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

// CompleteJob marks a job as completed
func (db *DB) CompleteJob(ctx context.Context, jobID, pdfLocation string) error {
	query := `
		UPDATE processing_jobs
		SET status = 'completed',
		    completed_at = NOW(),
		    pdf_location = $1
		WHERE id = $2
	`

	result, err := db.ExecContext(ctx, query, pdfLocation, jobID)
	if err != nil {
		return fmt.Errorf("failed to complete job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job not found: %s", jobID)
	}

	return nil
}

// FailJob marks a job as failed with error message
func (db *DB) FailJob(ctx context.Context, jobID, errorMessage string) error {
	query := `
		UPDATE processing_jobs
		SET status = 'failed',
		    error_message = $1
		WHERE id = $2
	`

	_, err := db.ExecContext(ctx, query, errorMessage, jobID)
	if err != nil {
		return fmt.Errorf("failed to mark job as failed: %w", err)
	}

	return nil
}
