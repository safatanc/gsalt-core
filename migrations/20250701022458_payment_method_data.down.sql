-- Add down migration script here

-- This script removes the seeded data from the `payment_methods` table.
TRUNCATE TABLE payment_methods RESTART IDENTITY;