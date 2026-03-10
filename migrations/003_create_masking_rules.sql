-- migrations/003_create_masking_rules.sql

CREATE TABLE IF NOT EXISTS masking_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID REFERENCES jobs(id) ON DELETE CASCADE,
    column_name VARCHAR NOT NULL,
    mask_type VARCHAR NOT NULL,
    parameters JSON,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
