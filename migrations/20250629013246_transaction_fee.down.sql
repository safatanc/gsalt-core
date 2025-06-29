-- Remove constraints first
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_total_amount_consistency;

ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_total_amount_gsalt_units_non_negative;

ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_fee_gsalt_units_non_negative;

-- Remove indexes
DROP INDEX IF EXISTS idx_transactions_total_amount_gsalt_units;

DROP INDEX IF EXISTS idx_transactions_fee_gsalt_units;

-- Remove columns
ALTER TABLE transactions
DROP COLUMN IF EXISTS total_amount_gsalt_units;

ALTER TABLE transactions DROP COLUMN IF EXISTS fee_gsalt_units;