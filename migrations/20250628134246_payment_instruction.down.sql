-- Add down migration script here

-- Remove payment_instructions column from transactions table

ALTER TABLE transactions DROP COLUMN payment_instructions;