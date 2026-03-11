-- migrations/001_create_jobs.sql

CREATE TABLE IF NOT EXISTS jobs (
    id BIGINT PRIMARY KEY DEFAULT (UUID_SHORT()),
    uuid VARCHAR(36) DEFAULT (UUID()),
    name VARCHAR NOT NULL,
    source_db_type VARCHAR,
    output_type VARCHAR,
    target_db_type VARCHAR,
    target_table_name VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) COMMENT='Masking jobs - passwords are not stored; target_db_* columns replace staging';
