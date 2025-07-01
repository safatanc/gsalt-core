-- Add up migration script here

-- This script creates the `payment_methods` table to store configuration for various payment gateways.
-- This approach moves the configuration from hard-coded maps in the service layer to a flexible, database-driven setup.

CREATE TABLE IF NOT EXISTS payment_methods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name VARCHAR(255) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    currency VARCHAR(3) NOT NULL,
    method_type VARCHAR(50) NOT NULL,
    provider_code VARCHAR(50) NOT NULL,
    provider_method_code VARCHAR(50) NOT NULL,
    provider_method_type VARCHAR(50) NOT NULL,
    payment_fee_flat BIGINT NOT NULL DEFAULT 0,
    payment_fee_percent DECIMAL(8, 5) NOT NULL DEFAULT 0.0,
    withdrawal_fee_flat BIGINT NOT NULL DEFAULT 0,
    withdrawal_fee_percent DECIMAL(8, 5) NOT NULL DEFAULT 0.0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    is_available_for_topup BOOLEAN NOT NULL DEFAULT TRUE,
    is_available_for_withdrawal BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Add indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_payment_methods_code ON payment_methods (code);

CREATE INDEX IF NOT EXISTS idx_payment_methods_method_type ON payment_methods (method_type);

CREATE INDEX IF NOT EXISTS idx_payment_methods_provider_code ON payment_methods (provider_code);

-- Add comments for clarity
COMMENT ON TABLE payment_methods IS 'Stores configuration for payment methods, abstracting gateway-specific details.';

COMMENT ON COLUMN payment_methods.name IS 'Human-readable name for display purposes (e.g., "Mandiri Virtual Account").';

COMMENT ON COLUMN payment_methods.code IS 'Unique code used internally and in client-facing APIs (e.g., ''VA_MANDIRI'').';

COMMENT ON COLUMN payment_methods.currency IS 'Currency of the payment method (e.g., ''IDR'', ''USD'').';

COMMENT ON COLUMN payment_methods.method_type IS 'General type of the method (e.g., ''VIRTUAL_ACCOUNT'', ''EWALLET'').';

COMMENT ON COLUMN payment_methods.provider_code IS 'The payment gateway provider (e.g., ''FLIP'', ''XENDIT'').';

COMMENT ON COLUMN payment_methods.provider_method_code IS 'Provider-specific code for the payment method (e.g., ''mandiri'', ''qris'').';

COMMENT ON COLUMN payment_methods.provider_method_type IS 'The type required by the provider for this method (e.g., ''virtual_account'').';

COMMENT ON COLUMN payment_methods.payment_fee_flat IS 'Flat fee in the specified currency for payments/top-ups.';

COMMENT ON COLUMN payment_methods.payment_fee_percent IS 'Percentage fee for payments/top-ups (e.g., 0.007 for 0.7%).';

COMMENT ON COLUMN payment_methods.withdrawal_fee_flat IS 'Flat fee in the specified currency for withdrawals/disbursements.';

COMMENT ON COLUMN payment_methods.withdrawal_fee_percent IS 'Percentage fee for withdrawals/disbursements (e.g., 0.002 for 0.2%).';

COMMENT ON COLUMN payment_methods.is_active IS 'Overall flag to enable or disable this payment method.';

COMMENT ON COLUMN payment_methods.is_available_for_topup IS 'Is this method available for users to select for top-ups/payments?';

COMMENT ON COLUMN payment_methods.is_available_for_withdrawal IS 'Is this method available for users to select for withdrawals?';