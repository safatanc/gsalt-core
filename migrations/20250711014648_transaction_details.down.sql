-- Add columns back to transactions
ALTER TABLE transactions
ADD COLUMN payment_instructions TEXT,
ADD COLUMN external_payment_id VARCHAR(255),
ADD COLUMN processing_status VARCHAR(20) DEFAULT 'PENDING';

-- Drop new columns from transactions
ALTER TABLE transactions
DROP CONSTRAINT IF EXISTS chk_payment_status,
DROP COLUMN payment_status,
DROP COLUMN payment_status_description,
DROP COLUMN payment_initiated_at,
DROP COLUMN payment_completed_at,
DROP COLUMN payment_failed_at,
DROP COLUMN payment_expired_at;

-- Move data back from payment_details to transactions
UPDATE transactions t
SET
    payment_instructions = pd.raw_provider_response::text,
    external_payment_id = pd.provider_payment_id
FROM payment_details pd
WHERE
    t.id = pd.transaction_id;

-- Drop payment_accounts and payment_details tables
DROP TABLE IF EXISTS payment_accounts;

DROP TABLE IF EXISTS payment_details;