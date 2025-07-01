-- Add down migration script here

-- Drop indexes
DROP INDEX IF EXISTS idx_accounts_account_type;

DROP INDEX IF EXISTS idx_accounts_status;

DROP INDEX IF EXISTS idx_accounts_kyc_status;

DROP INDEX IF EXISTS idx_accounts_last_activity;

-- Drop columns from accounts table
ALTER TABLE accounts
DROP COLUMN IF EXISTS account_type,
DROP COLUMN IF EXISTS status,
DROP COLUMN IF EXISTS kyc_status,
DROP COLUMN IF EXISTS daily_limit,
DROP COLUMN IF EXISTS monthly_limit,
DROP COLUMN IF EXISTS last_activity_at;

-- Drop ENUM types
DROP TYPE IF EXISTS account_type;

DROP TYPE IF EXISTS account_status;

DROP TYPE IF EXISTS kyc_status;