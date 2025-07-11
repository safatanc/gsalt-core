-- Create payment_details table
CREATE TABLE payment_details (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    transaction_id UUID NOT NULL REFERENCES transactions (id),
    provider VARCHAR(50) NOT NULL,
    provider_payment_id VARCHAR(255),
    payment_url TEXT,
    qr_code TEXT,
    virtual_account_number VARCHAR(50),
    virtual_account_bank VARCHAR(50),
    retail_outlet_code VARCHAR(50),
    retail_payment_code VARCHAR(50),
    card_token VARCHAR(255),
    expiry_time TIMESTAMP WITH TIME ZONE,
    payment_time TIMESTAMP WITH TIME ZONE,
    provider_fee_amount BIGINT,
    status_history JSONB DEFAULT '[]',
    raw_provider_response JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create payment_accounts table
CREATE TABLE payment_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    account_id UUID NOT NULL REFERENCES accounts (connect_id),
    type VARCHAR(20) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    account_number VARCHAR(50) NOT NULL,
    account_name VARCHAR(255) NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    verification_time TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_payment_account_type CHECK (
        type IN ('BANK_ACCOUNT', 'EWALLET')
    )
);

-- Modify transactions table
-- 1. Add new columns
ALTER TABLE transactions
ADD COLUMN payment_status VARCHAR(20) DEFAULT 'PENDING',
ADD COLUMN payment_status_description TEXT,
ADD COLUMN payment_initiated_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN payment_completed_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN payment_failed_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN payment_expired_at TIMESTAMP WITH TIME ZONE;

-- 2. Add constraints
ALTER TABLE transactions
ADD CONSTRAINT chk_payment_status CHECK (
    payment_status IN (
        'PENDING',
        'WAITING_PAYMENT',
        'PROCESSING',
        'COMPLETED',
        'FAILED',
        'EXPIRED',
        'CANCELLED'
    )
);

-- Add indexes
CREATE INDEX idx_payment_details_transaction_id ON payment_details (transaction_id);

CREATE INDEX idx_payment_details_provider ON payment_details (provider);

CREATE INDEX idx_payment_details_provider_payment_id ON payment_details (provider_payment_id);

CREATE INDEX idx_payment_details_payment_time ON payment_details (payment_time);

CREATE INDEX idx_payment_accounts_account_id ON payment_accounts (account_id);

CREATE INDEX idx_payment_accounts_provider_account ON payment_accounts (provider, account_number);

CREATE INDEX idx_payment_accounts_active ON payment_accounts (is_active);

CREATE INDEX idx_transactions_payment_status ON transactions (payment_status);

CREATE INDEX idx_transactions_payment_completed_at ON transactions (payment_completed_at);

CREATE INDEX idx_transactions_payment_expired_at ON transactions (payment_expired_at);

-- Move payment instructions data from transactions to payment_details
INSERT INTO
    payment_details (
        transaction_id,
        provider,
        provider_payment_id,
        raw_provider_response,
        created_at,
        updated_at
    )
SELECT
    id,
    payment_method as provider,
    external_payment_id as provider_payment_id,
    payment_instructions::jsonb as raw_provider_response,
    created_at,
    updated_at
FROM transactions
WHERE
    payment_instructions IS NOT NULL;

-- Drop unused columns from transactions
ALTER TABLE transactions
DROP COLUMN payment_instructions,
DROP COLUMN external_payment_id,
DROP COLUMN processing_status;