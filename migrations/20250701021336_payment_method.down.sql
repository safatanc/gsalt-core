-- Add down migration script here

DROP INDEX IF EXISTS idx_payment_methods_code;

DROP INDEX IF EXISTS idx_payment_methods_method_type;

DROP INDEX IF EXISTS idx_payment_methods_provider_code;

DROP TABLE IF EXISTS payment_methods;