-- Add down migration script here
-- Rollback GSALT system changes

-- Drop validation trigger and function
DROP TRIGGER IF EXISTS trigger_validate_gsalt_transaction ON transactions;

DROP FUNCTION IF EXISTS validate_gsalt_transaction ();

-- Drop indexes
DROP INDEX IF EXISTS idx_transactions_amount_gsalt_units;

DROP INDEX IF EXISTS idx_transactions_payment_currency;

DROP INDEX IF EXISTS idx_transactions_payment_method;

-- Drop constraints
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_amount_gsalt_units_non_negative;

ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_exchange_rate_idr_positive;

ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_payment_fields_consistency;

-- Restore original currency default (assuming it was NULL or empty)
ALTER TABLE transactions ALTER COLUMN currency DROP DEFAULT;

-- Add back the amount column (restore legacy structure)
ALTER TABLE transactions ADD COLUMN amount DECIMAL(15, 2);

-- Remove GSALT system columns
ALTER TABLE transactions DROP COLUMN IF EXISTS amount_gsalt_units;

ALTER TABLE transactions DROP COLUMN IF EXISTS exchange_rate_idr;

ALTER TABLE transactions DROP COLUMN IF EXISTS payment_amount;

ALTER TABLE transactions DROP COLUMN IF EXISTS payment_currency;

ALTER TABLE transactions DROP COLUMN IF EXISTS payment_method;