-- Add up migration script here
-- Update transactions table to support GSALT system

-- Add new columns for GSALT system
ALTER TABLE transactions
ADD COLUMN amount_gsalt_units BIGINT NOT NULL DEFAULT 0,
ADD COLUMN exchange_rate_idr DECIMAL(10, 2) DEFAULT 1000.00,
ADD COLUMN payment_amount BIGINT,
ADD COLUMN payment_currency VARCHAR(5),
ADD COLUMN payment_method VARCHAR(50);

-- Add comments for clarity
COMMENT ON COLUMN transactions.amount_gsalt_units IS 'Amount in GSALT units (1 GSALT = 100 units for 2 decimal places)';

COMMENT ON COLUMN transactions.exchange_rate_idr IS 'Exchange rate: how many IDR = 1 GSALT (default 1000 IDR = 1 GSALT)';

COMMENT ON COLUMN transactions.payment_amount IS 'Actual payment amount in payment_currency (e.g., 50000 for IDR, 3.5 for USD)';

COMMENT ON COLUMN transactions.payment_currency IS 'Currency used for payment (IDR, USD, EUR, SGD, etc)';

COMMENT ON COLUMN transactions.payment_method IS 'Payment method used (QRIS, BANK_TRANSFER, CREDIT_CARD, etc)';

-- Remove the legacy amount column since no backward compatibility needed
ALTER TABLE transactions DROP COLUMN IF EXISTS amount;

-- Update currency default to GSALT
ALTER TABLE transactions ALTER COLUMN currency SET DEFAULT 'GSALT';

-- Add indexes for performance
CREATE INDEX idx_transactions_amount_gsalt_units ON transactions (amount_gsalt_units);

CREATE INDEX idx_transactions_payment_currency ON transactions (payment_currency);

CREATE INDEX idx_transactions_payment_method ON transactions (payment_method);

-- Add constraints
ALTER TABLE transactions
ADD CONSTRAINT chk_amount_gsalt_units_non_negative CHECK (amount_gsalt_units >= 0);

ALTER TABLE transactions
ADD CONSTRAINT chk_exchange_rate_idr_positive CHECK (
    exchange_rate_idr IS NULL
    OR exchange_rate_idr > 0
);

-- Add constraint for payment fields consistency
ALTER TABLE transactions
ADD CONSTRAINT chk_payment_fields_consistency CHECK (
    (
        payment_amount IS NULL
        AND payment_currency IS NULL
    )
    OR (
        payment_amount IS NOT NULL
        AND payment_currency IS NOT NULL
    )
);

-- Add validation for transaction types and GSALT amounts
CREATE OR REPLACE FUNCTION validate_gsalt_transaction()
RETURNS TRIGGER AS $$
BEGIN
    -- Ensure GIFT transactions have positive amounts
    IF NEW.type IN ('GIFT_IN', 'GIFT_OUT') AND NEW.amount_gsalt_units <= 0 THEN
        RAISE EXCEPTION 'Gift transactions must have positive amounts';
    END IF;
    
    -- Ensure payment transactions have positive amounts  
    IF NEW.type = 'PAYMENT' AND NEW.amount_gsalt_units <= 0 THEN
        RAISE EXCEPTION 'Payment transactions must have positive amounts';
    END IF;
    
    -- Ensure transfer transactions have positive amounts
    IF NEW.type IN ('TRANSFER_IN', 'TRANSFER_OUT') AND NEW.amount_gsalt_units <= 0 THEN
        RAISE EXCEPTION 'Transfer transactions must have positive amounts';
    END IF;
    
    -- Validate payment amount is positive when payment info is provided
    IF NEW.payment_amount IS NOT NULL AND NEW.payment_amount <= 0 THEN
        RAISE EXCEPTION 'Payment amount must be positive';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create validation trigger
CREATE TRIGGER trigger_validate_gsalt_transaction
    BEFORE INSERT OR UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION validate_gsalt_transaction();