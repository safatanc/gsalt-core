-- Add down migration script here
DROP INDEX idx_transactions_account_id;

DROP TABLE transactions;

DROP TYPE transaction_type;

DROP TYPE transaction_status;