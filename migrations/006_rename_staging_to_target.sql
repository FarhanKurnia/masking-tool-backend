-- Migration: rename staging_* to target_* and add password hash columns
-- Description: Adjust column names for clarity and introduce password hashes
-- This migration is primarily for users upgrading from an older schema
-- that still stored connection credentials. New installations (after
-- revision 001) no longer create any of the staging/source/target
-- connection columns, so most of the statements below are effectively
-- no-ops thanks to the IF EXISTS guards.
-- Migration Date: 2026-03-10

-- rename staging columns to target equivalents (works even if target already exists)
ALTER TABLE jobs
    CHANGE COLUMN IF EXISTS staging_db_type target_db_type VARCHAR,
    CHANGE COLUMN IF EXISTS staging_db_host target_db_host VARCHAR,
    CHANGE COLUMN IF EXISTS staging_db_port target_db_port INT,
    CHANGE COLUMN IF EXISTS staging_db_name target_db_name VARCHAR,
    CHANGE COLUMN IF EXISTS staging_db_user target_db_user VARCHAR,
    CHANGE COLUMN IF EXISTS staging_db_password target_db_password VARCHAR,
    CHANGE COLUMN IF EXISTS staging_db_password_encrypted target_db_password_encrypted LONGTEXT,
    CHANGE COLUMN IF EXISTS staging_table_name target_table_name VARCHAR;

-- keep compatibility by leaving staging_* fields but marked deprecated (no-op rename earlier)

-- add hash columns for easy verification or indexing
ALTER TABLE jobs
    ADD COLUMN IF NOT EXISTS source_db_password_hash VARCHAR(64) COMMENT 'SHA256 hash of source password',
    ADD COLUMN IF NOT EXISTS target_db_password_hash VARCHAR(64) COMMENT 'SHA256 hash of target password';

-- optional: index the hash fields for lookup
ALTER TABLE jobs
    ADD INDEX IF NOT EXISTS idx_source_db_password_hash (source_db_password_hash),
    ADD INDEX IF NOT EXISTS idx_target_db_password_hash (target_db_password_hash);
