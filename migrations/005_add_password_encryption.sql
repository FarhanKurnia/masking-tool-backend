-- migrations/005_add_password_encryption.sql
-- Add encryption columns to existing tables for password protection

-- Add encrypted password columns to jobs table if they don't exist
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS source_db_password_encrypted LONGTEXT COMMENT 'Encrypted source database password';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS target_db_type VARCHAR COMMENT 'Type of target database (mysql, postgres)';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS target_db_host VARCHAR;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS target_db_port INT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS target_db_name VARCHAR;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS target_db_user VARCHAR;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS target_db_password VARCHAR COMMENT 'Deprecated: use target_db_password_encrypted instead';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS target_db_password_encrypted LONGTEXT COMMENT 'Encrypted target database password';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS target_table_name VARCHAR;

-- Create an encryption_keys table for key management
CREATE TABLE IF NOT EXISTS encryption_keys (
    id BIGINT PRIMARY KEY DEFAULT (UUID_SHORT()),
    uuid VARCHAR(36) DEFAULT (UUID()),
    key_name VARCHAR(255) NOT NULL UNIQUE,
    key_hash VARCHAR(64) NOT NULL COMMENT 'SHA256 hash of the encryption key (never store actual key)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    rotated_at TIMESTAMP
) COMMENT='Manage encryption keys for password protection - key_name is used to identify which key was used';

-- Add index for faster lookups
CREATE INDEX IF NOT EXISTS idx_encryption_keys_key_name ON encryption_keys(key_name);

-- Add tracking for which encryption key was used
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS encryption_key_name VARCHAR(255) COMMENT 'Name of encryption key used for passwords';

-- (Note: target_db_* columns added above replaced previous staging names)
