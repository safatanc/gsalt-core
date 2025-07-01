-- Add up migration script here

-- First drop existing constraints to avoid conflicts
DO $$ 
BEGIN
    -- Drop accounts constraints if they exist
    ALTER TABLE accounts
        DROP CONSTRAINT IF EXISTS chk_balance_non_negative,
        DROP CONSTRAINT IF EXISTS chk_daily_limit_positive,
        DROP CONSTRAINT IF EXISTS chk_monthly_limit_positive,
        DROP CONSTRAINT IF EXISTS chk_account_status_valid,
        DROP CONSTRAINT IF EXISTS chk_kyc_status_valid,
        DROP CONSTRAINT IF EXISTS chk_account_type_valid;

    -- Drop transactions constraints if they exist
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

    -- Drop payment_methods constraints if they exist
    ALTER TABLE payment_methods
        DROP CONSTRAINT IF EXISTS chk_payment_fee_flat_non_negative,
        DROP CONSTRAINT IF EXISTS chk_payment_fee_percent_non_negative,
        DROP CONSTRAINT IF EXISTS chk_withdrawal_fee_flat_non_negative,
        DROP CONSTRAINT IF EXISTS chk_withdrawal_fee_percent_non_negative,
        DROP CONSTRAINT IF EXISTS chk_payment_method_currency_valid;

    -- Drop vouchers constraints if they exist
    ALTER TABLE vouchers
        DROP CONSTRAINT IF EXISTS chk_value_non_negative,
        DROP CONSTRAINT IF EXISTS chk_voucher_status_valid,
        DROP CONSTRAINT IF EXISTS chk_voucher_type_valid,
        DROP CONSTRAINT IF EXISTS chk_discount_percentage_valid,
        DROP CONSTRAINT IF EXISTS chk_max_redeem_count_positive,
        DROP CONSTRAINT IF EXISTS chk_current_redeem_count_non_negative;
END $$;

-- Add constraints to accounts table
ALTER TABLE accounts
-- Balance constraints
ADD CONSTRAINT chk_balance_non_negative CHECK (balance >= 0),
-- Daily and monthly limit constraints
ADD CONSTRAINT chk_daily_limit_positive CHECK (
    daily_limit IS NULL
    OR daily_limit > 0
),
ADD CONSTRAINT chk_monthly_limit_positive CHECK (
    monthly_limit IS NULL
    OR monthly_limit > 0
),
-- Status validation
ADD CONSTRAINT chk_account_status_valid CHECK (
    status IN (
        'ACTIVE',
        'SUSPENDED',
        'BLOCKED'
    )
),
ADD CONSTRAINT chk_kyc_status_valid CHECK (
    kyc_status IN (
        'UNVERIFIED',
        'PENDING',
        'VERIFIED',
        'REJECTED'
    )
),
-- Account type validation
ADD CONSTRAINT chk_account_type_valid CHECK (
    account_type IN ('PERSONAL', 'MERCHANT')
);

-- Add constraints to transactions table
ALTER TABLE transactions
-- Amount constraints
ADD CONSTRAINT chk_amount_gsalt_units_non_negative CHECK (amount_gsalt_units >= 0),
ADD CONSTRAINT chk_fee_gsalt_units_non_negative CHECK (fee_gsalt_units >= 0),
ADD CONSTRAINT chk_total_amount_gsalt_units_non_negative CHECK (total_amount_gsalt_units >= 0),
-- Status validation
ADD CONSTRAINT chk_transaction_status_valid CHECK (
    status IN (
        'PENDING',
        'PROCESSING',
        'COMPLETED',
        'FAILED',
        'CANCELLED'
    )
),
-- Type validation
ADD CONSTRAINT chk_transaction_type_valid CHECK (
    type IN (
        'TOPUP',
        'TRANSFER_IN',
        'TRANSFER_OUT',
        'PAYMENT',
        'WITHDRAWAL',
        'GIFT_IN',
        'GIFT_OUT'
    )
),
-- Currency validation
ADD CONSTRAINT chk_currency_valid CHECK (
    currency IN (
        'GSALT',
        'IDR',
        'USD',
        'EUR',
        'SGD'
    )
),
-- Payment currency validation
ADD CONSTRAINT chk_payment_currency_valid CHECK (
    payment_currency IS NULL
    OR payment_currency IN (
        'GSALT',
        'IDR',
        'USD',
        'EUR',
        'SGD'
    )
),
-- Payment amount validation
ADD CONSTRAINT chk_payment_amount_non_negative CHECK (
    payment_amount IS NULL
    OR payment_amount >= 0
),
-- Exchange rate validation
ADD CONSTRAINT chk_exchange_rate_positive CHECK (
    exchange_rate_idr IS NULL
    OR exchange_rate_idr > 0
);

-- Add constraints to payment_methods table
ALTER TABLE payment_methods
-- Fee constraints
ADD CONSTRAINT chk_payment_fee_flat_non_negative CHECK (payment_fee_flat >= 0),
ADD CONSTRAINT chk_payment_fee_percent_non_negative CHECK (payment_fee_percent >= 0),
ADD CONSTRAINT chk_withdrawal_fee_flat_non_negative CHECK (withdrawal_fee_flat >= 0),
ADD CONSTRAINT chk_withdrawal_fee_percent_non_negative CHECK (withdrawal_fee_percent >= 0),
-- Currency validation
ADD CONSTRAINT chk_payment_method_currency_valid CHECK (
    currency IN (
        'GSALT',
        'IDR',
        'USD',
        'EUR',
        'SGD'
    )
);

-- Add constraints to vouchers table
ALTER TABLE vouchers
-- Value constraints
ADD CONSTRAINT chk_value_non_negative CHECK (value >= 0),
-- Status validation
ADD CONSTRAINT chk_voucher_status_valid CHECK (
    status IN (
        'ACTIVE',
        'INACTIVE',
        'REDEEMED',
        'EXPIRED'
    )
),
-- Type validation
ADD CONSTRAINT chk_voucher_type_valid CHECK (
    type IN (
        'BALANCE',
        'LOYALTY_POINTS',
        'DISCOUNT'
    )
),
-- Discount percentage validation
ADD CONSTRAINT chk_discount_percentage_valid CHECK (
    discount_percentage IS NULL
    OR (
        discount_percentage > 0
        AND discount_percentage <= 100
    )
),
-- Usage limit validation
ADD CONSTRAINT chk_max_redeem_count_positive CHECK (max_redeem_count > 0),
-- Usage count validation
ADD CONSTRAINT chk_current_redeem_count_non_negative CHECK (current_redeem_count >= 0);

-- Add indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_transactions_account_type_status ON transactions(account_id, type, status);

CREATE INDEX IF NOT EXISTS idx_transactions_external_ref ON transactions (external_reference_id)
WHERE
    external_reference_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transactions_payment_method ON transactions (payment_method)
WHERE
    payment_method IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_transactions_completed_at ON transactions (completed_at)
WHERE
    completed_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts (status);

CREATE INDEX IF NOT EXISTS idx_accounts_kyc_status ON accounts (kyc_status);

CREATE INDEX IF NOT EXISTS idx_accounts_account_type ON accounts (account_type);

CREATE INDEX IF NOT EXISTS idx_vouchers_code ON vouchers (code);

CREATE INDEX IF NOT EXISTS idx_vouchers_status ON vouchers (status);

CREATE INDEX IF NOT EXISTS idx_vouchers_valid_until ON vouchers (valid_until)
WHERE
    valid_until IS NOT NULL;