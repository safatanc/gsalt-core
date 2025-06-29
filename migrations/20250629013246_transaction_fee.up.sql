-- Add fee and total amount columns for transaction fee tracking
ALTER TABLE transactions
ADD COLUMN fee_gsalt_units BIGINT NOT NULL DEFAULT 0,
ADD COLUMN total_amount_gsalt_units BIGINT NOT NULL DEFAULT 0;

-- Add comments for clarity
COMMENT ON COLUMN transactions.fee_gsalt_units IS 'Fee amount in GSALT units charged for external payments';

COMMENT ON COLUMN transactions.total_amount_gsalt_units IS 'Total amount including fees in GSALT units (amount + fee)';

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