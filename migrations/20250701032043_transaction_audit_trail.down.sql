-- Add down migration script here

-- Drop trigger
DROP TRIGGER IF EXISTS trg_transaction_status_change ON transactions;

-- Drop trigger function
DROP FUNCTION IF EXISTS log_transaction_status_change ();

-- Drop indexes
DROP INDEX IF EXISTS idx_transaction_status_history_transaction_id;

DROP INDEX IF EXISTS idx_transaction_status_history_created_at;

DROP INDEX IF EXISTS idx_audit_logs_table_record;

DROP INDEX IF EXISTS idx_audit_logs_changed_at;

DROP INDEX IF EXISTS idx_audit_logs_action;

-- Drop tables
DROP TABLE IF EXISTS transaction_status_history;

DROP TABLE IF EXISTS audit_logs;

-- Drop ENUM type
DROP TYPE IF EXISTS audit_action;