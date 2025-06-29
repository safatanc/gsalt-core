-- Remove constraints first
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_total_amount_consistency;

ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_total_amount_gsalt_units_non_negative;

ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_fee_gsalt_units_non_negative;

-- Remove indexes
DROP INDEX IF EXISTS idx_transactions_processing_status;

DROP INDEX IF EXISTS idx_transactions_external_payment_id;

DROP INDEX IF EXISTS idx_transactions_total_amount_gsalt_units;

DROP INDEX IF EXISTS idx_transactions_fee_gsalt_units;

-- Revert data type changes
ALTER TABLE transactions
ALTER COLUMN payment_instructions TYPE JSONB USING payment_instructions::jsonb;

ALTER TABLE transactions
ALTER COLUMN exchange_rate_idr TYPE DECIMAL(10, 2);

ALTER TABLE transactions ALTER COLUMN currency TYPE VARCHAR(5);

ALTER TABLE transactions
ALTER COLUMN payment_currency TYPE VARCHAR(5);

-- Remove added columns
ALTER TABLE transactions DROP COLUMN IF EXISTS processing_status;

ALTER TABLE transactions DROP COLUMN IF EXISTS external_payment_id;

ALTER TABLE transactions
DROP COLUMN IF EXISTS total_amount_gsalt_units;

ALTER TABLE transactions DROP COLUMN IF EXISTS fee_gsalt_units;