-- Add down migration script here

-- Drop indexes
DROP INDEX IF EXISTS idx_transactions_account_type_status;

DROP INDEX IF EXISTS idx_transactions_external_ref;

DROP INDEX IF EXISTS idx_transactions_payment_method;

DROP INDEX IF EXISTS idx_transactions_completed_at;

DROP INDEX IF EXISTS idx_accounts_status;

DROP INDEX IF EXISTS idx_accounts_kyc_status;

DROP INDEX IF EXISTS idx_accounts_account_type;

DROP INDEX IF EXISTS idx_vouchers_code;

DROP INDEX IF EXISTS idx_vouchers_status;

DROP INDEX IF EXISTS idx_vouchers_valid_until;

-- Drop constraints from accounts table
ALTER TABLE accounts
DROP CONSTRAINT IF EXISTS chk_balance_non_negative,
DROP CONSTRAINT IF EXISTS chk_daily_limit_positive,
DROP CONSTRAINT IF EXISTS chk_monthly_limit_positive,
DROP CONSTRAINT IF EXISTS chk_account_status_valid,
DROP CONSTRAINT IF EXISTS chk_kyc_status_valid,
DROP CONSTRAINT IF EXISTS chk_account_type_valid;

-- Drop constraints from transactions table
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_amount_gsalt_units_non_negative,
DROP CONSTRAINT IF EXISTS chk_fee_gsalt_units_non_negative,
DROP CONSTRAINT IF EXISTS chk_total_amount_gsalt_units_non_negative,
DROP CONSTRAINT IF EXISTS chk_transaction_status_valid,
DROP CONSTRAINT IF EXISTS chk_transaction_type_valid,
DROP CONSTRAINT IF EXISTS chk_currency_valid,
DROP CONSTRAINT IF EXISTS chk_payment_currency_valid,
DROP CONSTRAINT IF EXISTS chk_payment_amount_non_negative,
DROP CONSTRAINT IF EXISTS chk_exchange_rate_positive;

-- Drop constraints from payment_methods table
ALTER TABLE payment_methods
DROP CONSTRAINT IF EXISTS chk_payment_fee_flat_non_negative,
DROP CONSTRAINT IF EXISTS chk_payment_fee_percent_non_negative,
DROP CONSTRAINT IF EXISTS chk_withdrawal_fee_flat_non_negative,
DROP CONSTRAINT IF EXISTS chk_withdrawal_fee_percent_non_negative,
DROP CONSTRAINT IF EXISTS chk_payment_method_currency_valid;

-- Drop constraints from vouchers table
ALTER TABLE vouchers
DROP CONSTRAINT IF EXISTS chk_value_non_negative,
DROP CONSTRAINT IF EXISTS chk_voucher_status_valid,
DROP CONSTRAINT IF EXISTS chk_voucher_type_valid,
DROP CONSTRAINT IF EXISTS chk_discount_percentage_valid,
DROP CONSTRAINT IF EXISTS chk_max_redeem_count_positive,
DROP CONSTRAINT IF EXISTS chk_current_redeem_count_non_negative;