-- Add down migration script here

-- Drop function first
DROP FUNCTION IF EXISTS refresh_all_materialized_views ();

-- Drop materialized views and their indexes
DROP MATERIALIZED VIEW IF EXISTS mv_payment_method_performance;

DROP MATERIALIZED VIEW IF EXISTS mv_merchant_analytics;

DROP MATERIALIZED VIEW IF EXISTS mv_voucher_analytics;

-- Restore original refresh function
CREATE OR REPLACE FUNCTION refresh_all_materialized_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_transaction_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_account_balance_summary;
END;
$$ LANGUAGE plpgsql;