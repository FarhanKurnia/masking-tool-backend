-- migrations/001_create_jobs.sql

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR NOT NULL,
    source_db_host VARCHAR,
    source_db_port INT,
    source_db_name VARCHAR,
    source_db_user VARCHAR,
    source_db_password VARCHAR,
    output_type VARCHAR,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);
