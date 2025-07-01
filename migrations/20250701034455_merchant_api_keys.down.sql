-- Add down migration script here

-- Drop function first
DROP FUNCTION IF EXISTS refresh_all_materialized_views ();

-- Drop materialized view and its index
DROP MATERIALIZED VIEW IF EXISTS mv_merchant_api_key_analytics;

-- Drop audit table and its indexes
DROP INDEX IF EXISTS idx_merchant_api_key_usage_api_key;

DROP INDEX IF EXISTS idx_merchant_api_key_usage_created;

DROP TABLE IF EXISTS merchant_api_key_usage;

-- Drop merchant_api_keys table and its indexes
DROP INDEX IF EXISTS idx_merchant_api_keys_merchant_id;

DROP INDEX IF EXISTS idx_merchant_api_keys_prefix;

DROP INDEX IF EXISTS idx_merchant_api_keys_last_used;

DROP INDEX IF EXISTS idx_merchant_api_keys_expires;

DROP TABLE IF EXISTS merchant_api_keys;

-- Drop API key scope enum
DROP TYPE IF EXISTS api_key_scope;

-- Restore original refresh function
CREATE OR REPLACE FUNCTION refresh_all_materialized_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_transaction_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_account_balance_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_payment_method_performance;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_merchant_analytics;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_voucher_analytics;
END;
$$ LANGUAGE plpgsql;