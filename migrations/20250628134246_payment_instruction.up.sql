-- Add up migration script here

-- Add payment_instructions column to transactions table
-- This column will store payment instructions from Xendit as JSON

ALTER TABLE transactions ADD COLUMN payment_instructions JSONB;

-- Add comment to explain the column
COMMENT ON COLUMN transactions.payment_instructions IS 'JSON field containing payment instructions from Xendit (QR code, VA number, checkout URL, etc.)';