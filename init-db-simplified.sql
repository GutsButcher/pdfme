-- ============================================================
-- PDF Generation System - Simplified Database Schema
-- Job State Tracking Only (No Content Storage)
-- ============================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- Processing Jobs (Job State Tracking)
-- ============================================================
CREATE TABLE processing_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- File identification (UNIQUE constraint prevents duplicates)
    file_hash VARCHAR(64) UNIQUE NOT NULL,  -- S3 ETag (MD5)
    filename VARCHAR(255) NOT NULL,

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- Values: pending, processing, completed, failed

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    processing_started_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Storage location (S3 path after completion)
    pdf_location TEXT,

    -- Error handling
    error_message TEXT,
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3
);

-- Indexes for common queries
CREATE INDEX idx_jobs_file_hash ON processing_jobs(file_hash);
CREATE INDEX idx_jobs_status ON processing_jobs(status);
CREATE INDEX idx_jobs_processing_timeout ON processing_jobs(processing_started_at)
    WHERE status = 'processing';

-- ============================================================
-- Helper Functions
-- ============================================================

-- Function to find stuck jobs (processing > 1 hour)
CREATE OR REPLACE FUNCTION find_stuck_jobs()
RETURNS TABLE (
    job_id UUID,
    filename VARCHAR(255),
    processing_started_at TIMESTAMP,
    minutes_stuck NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        id,
        processing_jobs.filename,
        processing_jobs.processing_started_at,
        EXTRACT(EPOCH FROM (NOW() - processing_jobs.processing_started_at)) / 60 as minutes_stuck
    FROM processing_jobs
    WHERE status = 'processing'
      AND processing_started_at < NOW() - INTERVAL '1 hour'
      AND retry_count < max_retries
    ORDER BY processing_started_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to find stuck pending jobs (pending > 10 minutes)
CREATE OR REPLACE FUNCTION find_stuck_pending_jobs()
RETURNS TABLE (
    job_id UUID,
    filename VARCHAR(255),
    file_hash VARCHAR(64),
    created_at TIMESTAMP,
    minutes_pending NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        id,
        processing_jobs.filename,
        processing_jobs.file_hash,
        processing_jobs.created_at,
        EXTRACT(EPOCH FROM (NOW() - processing_jobs.created_at)) / 60 as minutes_pending
    FROM processing_jobs
    WHERE status = 'pending'
      AND created_at < NOW() - INTERVAL '10 minutes'
      AND retry_count < max_retries
    ORDER BY created_at ASC;
END;
$$ LANGUAGE plpgsql;

-- Function to mark job for retry
CREATE OR REPLACE FUNCTION mark_job_for_retry(job_uuid UUID)
RETURNS BOOLEAN AS $$
DECLARE
    current_retries INT;
BEGIN
    -- Get current retry count
    SELECT retry_count INTO current_retries
    FROM processing_jobs
    WHERE id = job_uuid;

    -- Check if we've exceeded max retries
    IF current_retries >= 3 THEN
        -- Mark as failed
        UPDATE processing_jobs
        SET status = 'failed',
            error_message = 'Max retries exceeded'
        WHERE id = job_uuid;
        RETURN FALSE;
    ELSE
        -- Reset to pending for retry
        UPDATE processing_jobs
        SET status = 'pending',
            retry_count = retry_count + 1,
            processing_started_at = NULL
        WHERE id = job_uuid;
        RETURN TRUE;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old completed jobs (90 days)
CREATE OR REPLACE FUNCTION cleanup_old_jobs(days_to_keep INT DEFAULT 90)
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM processing_jobs
    WHERE status = 'completed'
      AND completed_at < NOW() - (days_to_keep || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- Helpful Views
-- ============================================================

-- View for job statistics
CREATE VIEW job_statistics AS
SELECT
    status,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (completed_at - created_at))) as avg_duration_seconds,
    MIN(created_at) as oldest_job,
    MAX(created_at) as newest_job
FROM processing_jobs
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY status;

-- View for failed jobs
CREATE VIEW failed_jobs AS
SELECT
    id,
    filename,
    file_hash,
    error_message,
    retry_count,
    created_at,
    processing_started_at
FROM processing_jobs
WHERE status = 'failed'
ORDER BY created_at DESC;

-- ============================================================
-- Database Initialization Complete
-- ============================================================

-- Log database version
CREATE TABLE schema_version (
    version INT PRIMARY KEY,
    description TEXT,
    applied_at TIMESTAMP DEFAULT NOW()
);

INSERT INTO schema_version (version, description)
VALUES (3, 'Added stuck pending jobs detection - prevents jobs stuck in pending status');

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO pdfme;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO pdfme;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO pdfme;

-- ============================================================
-- Sample Queries for Operations
-- ============================================================

-- Find stuck jobs (processing > 1 hour):
-- SELECT * FROM find_stuck_jobs();

-- Find stuck pending jobs (pending > 10 minutes):
-- SELECT * FROM find_stuck_pending_jobs();

-- Retry a stuck job:
-- SELECT mark_job_for_retry('job-uuid-here');

-- View statistics:
-- SELECT * FROM job_statistics;

-- View failed jobs:
-- SELECT * FROM failed_jobs;

-- Cleanup old jobs:
-- SELECT cleanup_old_jobs(90);
