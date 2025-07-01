-- Add up migration script here

-- Create materialized view for payment method performance
CREATE MATERIALIZED VIEW mv_payment_method_performance AS
SELECT
    t.payment_method,
    pm.name as payment_method_name,
    pm.method_type,
    pm.provider_code,
    COUNT(*) as total_transactions,
    COUNT(
        CASE
            WHEN t.status = 'COMPLETED' THEN 1
        END
    ) as completed_transactions,
    COUNT(
        CASE
            WHEN t.status = 'FAILED' THEN 1
        END
    ) as failed_transactions,
    ROUND(
        COUNT(
            CASE
                WHEN t.status = 'COMPLETED' THEN 1
            END
        )::numeric / COUNT(*)::numeric * 100,
        2
    ) as success_rate,
    SUM(t.amount_gsalt_units) as total_amount_gsalt_units,
    SUM(t.fee_gsalt_units) as total_fee_gsalt_units,
    AVG(
        EXTRACT(
            EPOCH
            FROM (t.completed_at - t.created_at)
        )
    ) as avg_processing_time_seconds
FROM
    transactions t
    LEFT JOIN payment_methods pm ON t.payment_method = pm.code
WHERE
    t.payment_method IS NOT NULL
GROUP BY
    t.payment_method,
    pm.name,
    pm.method_type,
    pm.provider_code;

-- Create unique index for materialized view refresh
CREATE UNIQUE INDEX idx_mv_payment_method_performance ON mv_payment_method_performance (payment_method);

-- Create materialized view for merchant analytics
CREATE MATERIALIZED VIEW mv_merchant_analytics AS
WITH
    merchant_transactions AS (
        SELECT
            a.connect_id as merchant_id,
            t.type,
            t.status,
            t.amount_gsalt_units,
            t.fee_gsalt_units,
            t.total_amount_gsalt_units,
            DATE_TRUNC('month', t.created_at) as transaction_month
        FROM transactions t
            JOIN accounts a ON t.account_id = a.connect_id
        WHERE
            a.account_type = 'MERCHANT'
    )
SELECT
    merchant_id,
    transaction_month,
    COUNT(*) as total_transactions,
    COUNT(
        CASE
            WHEN status = 'COMPLETED' THEN 1
        END
    ) as completed_transactions,
    SUM(amount_gsalt_units) as total_amount_gsalt_units,
    SUM(fee_gsalt_units) as total_fee_gsalt_units,
    SUM(total_amount_gsalt_units) as total_revenue_gsalt_units,
    COUNT(
        CASE
            WHEN type = 'PAYMENT' THEN 1
        END
    ) as payment_count,
    COUNT(
        CASE
            WHEN type = 'WITHDRAWAL' THEN 1
        END
    ) as withdrawal_count
FROM merchant_transactions
GROUP BY
    merchant_id,
    transaction_month;

-- Create unique index for materialized view refresh
CREATE UNIQUE INDEX idx_mv_merchant_analytics ON mv_merchant_analytics (
    merchant_id,
    transaction_month
);

-- Create materialized view for voucher analytics
CREATE MATERIALIZED VIEW mv_voucher_analytics AS
WITH
    voucher_usage AS (
        SELECT
            v.id as voucher_id,
            v.code as voucher_code,
            v.name as voucher_name,
            v.type as voucher_type,
            v.value,
            v.currency,
            v.max_redeem_count,
            v.current_redeem_count,
            v.valid_from,
            v.valid_until,
            vr.redeemed_at,
            t.amount_gsalt_units as transaction_amount
        FROM
            vouchers v
            LEFT JOIN voucher_redemptions vr ON v.id = vr.voucher_id
            LEFT JOIN transactions t ON vr.transaction_id = t.id
    )
SELECT
    voucher_id,
    voucher_code,
    voucher_name,
    voucher_type,
    value,
    currency,
    max_redeem_count,
    current_redeem_count,
    ROUND(
        current_redeem_count::numeric / max_redeem_count::numeric * 100,
        2
    ) as redemption_rate,
    COUNT(redeemed_at) as total_redemptions,
    SUM(transaction_amount) as total_transaction_amount,
    MIN(redeemed_at) as first_redemption,
    MAX(redeemed_at) as last_redemption,
    NOW() - valid_from as age,
    CASE
        WHEN valid_until IS NOT NULL THEN CASE
            WHEN NOW() > valid_until THEN 'EXPIRED'
            ELSE CONCAT(
                EXTRACT(
                    DAY
                    FROM valid_until - NOW()
                )::integer,
                ' days left'
            )
        END
        ELSE 'NO EXPIRY'
    END as expiry_status
FROM voucher_usage
GROUP BY
    voucher_id,
    voucher_code,
    voucher_name,
    voucher_type,
    value,
    currency,
    max_redeem_count,
    current_redeem_count,
    valid_from,
    valid_until;

-- Create unique index for materialized view refresh
CREATE UNIQUE INDEX idx_mv_voucher_analytics ON mv_voucher_analytics (voucher_id);

-- Update refresh function to include new views
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