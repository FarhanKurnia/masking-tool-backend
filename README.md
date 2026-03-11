# masking-tool-backend

## Password Encryption

This backend supports AES-256-GCM encryption of database credentials stored in
`jobs` table. To enable encryption, set the following environment variables:

- `ENCRYPTION_KEY_NAME`: name of the key to use (default `default`).
- `ENCRYPTION_KEY_<KEY_NAME>`: base64-encoded encryption key. Must be 16, 24 or
  32 bytes after decoding.

Example:

```bash
export ENCRYPTION_KEY_NAME=default
export ENCRYPTION_KEY_default=$(openssl rand -base64 32)
```

When the app starts it initializes an encryption manager. If no key is provided,
passwords will be stored in plaintext columns for compatibility. The migration
`005_add_password_encryption.sql` adds encrypted columns and the
`encryption_keys` table for key metadata.

**Note:** as of version 2026 the system uses `source_db_*` and `target_db_*`
columns (formerly `staging_db_*`) to clarify the relationship; migration
`006_rename_staging_to_target.sql` performs the rename and adds SHA‑256 hash
fields for both credentials.

**Security update:** connection parameters (host, port, database, user,
and password) are no longer stored in the `jobs` table – only the
`source_db_type`, `target_db_type` and other non‑sensitive metadata are
recorded. Credentials must be supplied each time a job is executed, so the
service never persists them.
