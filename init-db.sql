-- ============================================================
-- PDF Generation System - Database Schema
-- ============================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- Job Tracking (Central to all services)
-- ============================================================
CREATE TABLE processing_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- File information
    org_id VARCHAR(50) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    file_hash VARCHAR(64) UNIQUE NOT NULL,  -- SHA256 for deduplication

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'uploaded',
    current_stage VARCHAR(100),

    -- Stage timestamps
    uploaded_at TIMESTAMP DEFAULT NOW(),
    parsing_started_at TIMESTAMP,
    parsed_at TIMESTAMP,
    generating_started_at TIMESTAMP,
    generated_at TIMESTAMP,
    storing_started_at TIMESTAMP,
    stored_at TIMESTAMP,
    completed_at TIMESTAMP,

    -- Error handling
    error_message TEXT,
    error_stage VARCHAR(50),
    retry_count INT DEFAULT 0,
    max_retries INT DEFAULT 3,

    -- Metadata
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for processing_jobs
CREATE INDEX idx_jobs_status ON processing_jobs(status);
CREATE INDEX idx_jobs_org_created ON processing_jobs(org_id, created_at DESC);
CREATE INDEX idx_jobs_file_hash ON processing_jobs(file_hash);
CREATE INDEX idx_jobs_updated ON processing_jobs(updated_at);

-- ============================================================
-- Parser Domain
-- ============================================================
CREATE TABLE parsed_statements (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key to job
    job_id UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,

    -- Organization and file info
    org_id VARCHAR(50) NOT NULL,
    filename VARCHAR(255) NOT NULL,

    -- Parsed statement data
    account_name VARCHAR(255),
    card_number VARCHAR(50),
    statement_date DATE,
    available_balance DECIMAL(15,2),

    -- Parser metadata
    parsed_at TIMESTAMP DEFAULT NOW(),
    parser_version VARCHAR(20) DEFAULT '1.0'
);

-- Indexes for parsed_statements
CREATE INDEX idx_statements_job ON parsed_statements(job_id);
CREATE INDEX idx_statements_org_date ON parsed_statements(org_id, statement_date DESC);
CREATE INDEX idx_statements_parsed_at ON parsed_statements(parsed_at);

CREATE TABLE transactions (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key to statement
    statement_id BIGINT NOT NULL REFERENCES parsed_statements(id) ON DELETE CASCADE,

    -- Transaction data
    transaction_date DATE,
    post_date DATE,
    description TEXT,
    amount DECIMAL(15,2),
    is_credit BOOLEAN
);

-- Indexes for transactions
CREATE INDEX idx_transactions_statement ON transactions(statement_id);
CREATE INDEX idx_transactions_date ON transactions(transaction_date);

-- ============================================================
-- Template Domain (PDF Generator)
-- ============================================================
CREATE TABLE templates (
    id BIGSERIAL PRIMARY KEY,

    -- Template identification
    name VARCHAR(100) NOT NULL,
    version INT NOT NULL DEFAULT 1,

    -- Template metadata
    org_id VARCHAR(50),  -- NULL = global template
    description TEXT,
    items_per_page INT DEFAULT 15,

    -- Template storage (JSON stored in file system, path here)
    file_path TEXT NOT NULL,

    -- Status
    is_active BOOLEAN DEFAULT true,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    -- Constraints
    UNIQUE (name, version)
);

-- Indexes for templates
CREATE INDEX idx_templates_active ON templates(name, is_active);
CREATE INDEX idx_templates_org ON templates(org_id);

-- ============================================================
-- Storage Domain
-- ============================================================
CREATE TABLE stored_files (
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key to job
    job_id UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,

    -- MinIO storage information
    bucket_name VARCHAR(100) NOT NULL,
    object_key TEXT NOT NULL,  -- Full path in MinIO
    file_size BIGINT,
    content_type VARCHAR(100) DEFAULT 'application/pdf',

    -- Metadata
    stored_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for stored_files
CREATE INDEX idx_stored_job ON stored_files(job_id);
CREATE INDEX idx_stored_bucket_key ON stored_files(bucket_name, object_key);
CREATE INDEX idx_stored_at ON stored_files(stored_at);

-- ============================================================
-- Outbox Pattern (Reliable Messaging)
-- ============================================================
CREATE TABLE outbox_messages (
    id BIGSERIAL PRIMARY KEY,

    -- Message identification
    job_id UUID NOT NULL REFERENCES processing_jobs(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,  -- file_uploaded, statement_parsed, pdf_generated, file_stored

    -- Target queue
    queue_name VARCHAR(100) NOT NULL,  -- parse_ready, pdf_ready, storage_ready

    -- Message payload
    payload JSONB NOT NULL,

    -- Delivery status
    sent BOOLEAN DEFAULT false,
    sent_at TIMESTAMP,
    retry_count INT DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for outbox_messages
CREATE INDEX idx_outbox_unsent ON outbox_messages(created_at) WHERE sent = false;
CREATE INDEX idx_outbox_job ON outbox_messages(job_id);

-- ============================================================
-- Triggers for updated_at
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_jobs_updated_at BEFORE UPDATE ON processing_jobs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================
-- Insert default template
-- ============================================================
INSERT INTO templates (name, version, org_id, description, items_per_page, file_path, is_active)
VALUES ('new-template', 1, NULL, 'Default statement template', 15, 'templates/new-template.json', true);

-- ============================================================
-- Helpful Views
-- ============================================================

-- View for job statistics
CREATE VIEW job_statistics AS
SELECT
    status,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (completed_at - uploaded_at))) as avg_duration_seconds,
    MIN(created_at) as oldest_job,
    MAX(created_at) as newest_job
FROM processing_jobs
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY status;

-- View for failed jobs
CREATE VIEW failed_jobs AS
SELECT
    id,
    org_id,
    filename,
    status,
    error_message,
    error_stage,
    retry_count,
    created_at,
    updated_at
FROM processing_jobs
WHERE status = 'failed'
ORDER BY created_at DESC;

-- View for stuck jobs (processing > 10 minutes)
CREATE VIEW stuck_jobs AS
SELECT
    id,
    org_id,
    filename,
    status,
    current_stage,
    updated_at,
    EXTRACT(EPOCH FROM (NOW() - updated_at)) / 60 as minutes_stuck
FROM processing_jobs
WHERE status NOT IN ('completed', 'failed')
  AND updated_at < NOW() - INTERVAL '10 minutes'
ORDER BY updated_at ASC;

-- ============================================================
-- Maintenance Functions
-- ============================================================

-- Function to clean up old completed jobs
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

-- Function to clean up old sent outbox messages
CREATE OR REPLACE FUNCTION cleanup_old_outbox(days_to_keep INT DEFAULT 7)
RETURNS INT AS $$
DECLARE
    deleted_count INT;
BEGIN
    DELETE FROM outbox_messages
    WHERE sent = true
      AND sent_at < NOW() - (days_to_keep || ' days')::INTERVAL;

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

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
VALUES (1, 'Initial schema with job tracking, parsed statements, templates, storage, and outbox pattern');

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO pdfme;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO pdfme;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO pdfme;
