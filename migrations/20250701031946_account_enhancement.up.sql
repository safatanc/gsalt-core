-- Add up migration script here

-- Create ENUM types for account statuses
CREATE TYPE account_type AS ENUM ('PERSONAL', 'MERCHANT');

CREATE TYPE account_status AS ENUM ('ACTIVE', 'SUSPENDED', 'BLOCKED');

CREATE TYPE kyc_status AS ENUM ('UNVERIFIED', 'PENDING', 'VERIFIED', 'REJECTED');

-- Add new columns to accounts table
ALTER TABLE accounts
ADD COLUMN account_type account_type NOT NULL DEFAULT 'PERSONAL',
ADD COLUMN status account_status NOT NULL DEFAULT 'ACTIVE',
ADD COLUMN kyc_status kyc_status NOT NULL DEFAULT 'UNVERIFIED',
ADD COLUMN daily_limit bigint,
ADD COLUMN monthly_limit bigint,
ADD COLUMN last_activity_at timestamp with time zone,
-- Add constraints for limits
ADD CONSTRAINT chk_daily_limit_positive CHECK (
    daily_limit IS NULL
    OR daily_limit > 0
),
ADD CONSTRAINT chk_monthly_limit_positive CHECK (
    monthly_limit IS NULL
    OR monthly_limit > 0
),
ADD CONSTRAINT chk_monthly_limit_greater_daily CHECK (
    monthly_limit IS NULL
    OR daily_limit IS NULL
    OR monthly_limit >= daily_limit
);

-- Create index for frequently accessed columns
CREATE INDEX idx_accounts_account_type ON accounts (account_type);

CREATE INDEX idx_accounts_status ON accounts (status);

CREATE INDEX idx_accounts_kyc_status ON accounts (kyc_status);

CREATE INDEX idx_accounts_last_activity ON accounts (last_activity_at);