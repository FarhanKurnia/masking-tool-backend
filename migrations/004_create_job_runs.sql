-- migrations/004_create_job_runs.sql

CREATE TABLE IF NOT EXISTS job_runs (
    id BIGINT PRIMARY KEY DEFAULT (UUID_SHORT()),
    uuid VARCHAR(36) DEFAULT (UUID()),
    job_id BIGINT REFERENCES jobs(id) ON DELETE CASCADE,
    status VARCHAR,
    rows_processed INT DEFAULT 0,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    log TEXT
);
