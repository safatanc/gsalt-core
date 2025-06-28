-- Add down migration script here
DROP INDEX idx_vouchers_valid_until;

DROP INDEX idx_vouchers_status;

DROP INDEX idx_vouchers_code;

DROP TABLE vouchers;

DROP TYPE voucher_status;

DROP TYPE voucher_type;