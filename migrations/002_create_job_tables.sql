-- migrations/002_create_job_tables.sql

CREATE TABLE IF NOT EXISTS job_tables (
    id BIGINT PRIMARY KEY DEFAULT (UUID_SHORT()),
    uuid VARCHAR(36) DEFAULT (UUID()),
    job_id BIGINT REFERENCES jobs(id) ON DELETE CASCADE,
    table_name VARCHAR NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
