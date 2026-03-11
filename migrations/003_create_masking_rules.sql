-- migrations/003_create_masking_rules.sql

CREATE TABLE IF NOT EXISTS masking_rules (
    id BIGINT PRIMARY KEY DEFAULT (UUID_SHORT()),
    uuid VARCHAR(36) DEFAULT (UUID()),
    job_id BIGINT REFERENCES jobs(id) ON DELETE CASCADE,
    column_name VARCHAR NOT NULL,
    mask_type VARCHAR NOT NULL,
    parameters JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
