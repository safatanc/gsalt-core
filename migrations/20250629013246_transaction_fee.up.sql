-- Add fee and total amount columns for transaction fee tracking
ALTER TABLE transactions
ADD COLUMN fee_gsalt_units BIGINT NOT NULL DEFAULT 0,
ADD COLUMN total_amount_gsalt_units BIGINT NOT NULL DEFAULT 0;

-- Add missing columns that exist in the model but not in database
ALTER TABLE transactions
ADD COLUMN external_payment_id VARCHAR(255),
ADD COLUMN processing_status VARCHAR(20) DEFAULT 'PENDING';

-- Update transaction_status enum to include PROCESSING
ALTER TYPE transaction_status ADD VALUE IF NOT EXISTS 'PROCESSING';

-- Update transaction_type enum to include WITHDRAWAL
ALTER TYPE transaction_type ADD VALUE IF NOT EXISTS 'WITHDRAWAL';

-- Fix data type for exchange_rate_idr to match model (decimal(20,4))
ALTER TABLE transactions
ALTER COLUMN exchange_rate_idr TYPE DECIMAL(20, 4);

-- Change payment_instructions from JSONB to TEXT to match model
ALTER TABLE transactions ALTER COLUMN payment_instructions TYPE TEXT;

-- Update currency column to match model (varchar(10))
ALTER TABLE transactions ALTER COLUMN currency TYPE VARCHAR(10);

-- Update payment_currency column to match model (varchar(10))
ALTER TABLE transactions
ALTER COLUMN payment_currency TYPE VARCHAR(10);

-- Add comments for clarity
COMMENT ON COLUMN transactions.fee_gsalt_units IS 'Fee amount in GSALT units charged for external payments';

COMMENT ON COLUMN transactions.total_amount_gsalt_units IS 'Total amount including fees in GSALT units (amount + fee)';

COMMENT ON COLUMN transactions.external_payment_id IS 'External payment provider payment ID (e.g., Flip bill payment ID)';

COMMENT ON COLUMN transactions.processing_status IS 'Processing status for complex transactions';

-- Update existing records to set total_amount_gsalt_units = amount_gsalt_units + fee_gsalt_units
-- This ensures data consistency before adding constraints
UPDATE transactions
SET
    total_amount_gsalt_units = amount_gsalt_units + fee_gsalt_units
WHERE
    total_amount_gsalt_units = 0;

-- Add indexes for performance
CREATE INDEX idx_transactions_fee_gsalt_units ON transactions (fee_gsalt_units);

CREATE INDEX idx_transactions_total_amount_gsalt_units ON transactions (total_amount_gsalt_units);

CREATE INDEX idx_transactions_external_payment_id ON transactions (external_payment_id);

CREATE INDEX idx_transactions_processing_status ON transactions (processing_status);

-- Add constraints
ALTER TABLE transactions
ADD CONSTRAINT chk_fee_gsalt_units_non_negative CHECK (fee_gsalt_units >= 0);

ALTER TABLE transactions
ADD CONSTRAINT chk_total_amount_gsalt_units_non_negative CHECK (total_amount_gsalt_units >= 0);

-- Add constraint to ensure total amount consistency (total >= amount)
ALTER TABLE transactions
ADD CONSTRAINT chk_total_amount_consistency CHECK (
    total_amount_gsalt_units >= amount_gsalt_units
);