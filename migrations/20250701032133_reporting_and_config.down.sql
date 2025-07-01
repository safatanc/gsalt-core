-- Add down migration script here

-- Drop function
DROP FUNCTION IF EXISTS refresh_all_materialized_views ();

-- Drop materialized views and their indexes
DROP MATERIALIZED VIEW IF EXISTS mv_daily_transaction_summary;

DROP MATERIALIZED VIEW IF EXISTS mv_account_balance_summary;

-- Drop system configurations table and index
DROP INDEX IF EXISTS idx_system_configurations_category_key;

DROP TABLE IF EXISTS system_configurations;