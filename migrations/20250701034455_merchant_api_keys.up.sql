-- Add up migration script here

-- Create type for API key scopes
CREATE TYPE api_key_scope AS ENUM (
    'READ',
    'WRITE',
    'PAYMENT',
    'WITHDRAWAL',
    'ADMIN'
);

-- Create merchant_api_keys table
CREATE TABLE merchant_api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    merchant_id UUID NOT NULL REFERENCES accounts (connect_id),
    key_name VARCHAR(100) NOT NULL,
    api_key VARCHAR(255) NOT NULL UNIQUE,
    prefix VARCHAR(10) NOT NULL,
    scopes api_key_scope[] NOT NULL,
    rate_limit INTEGER NOT NULL DEFAULT 100,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT chk_rate_limit_positive CHECK (rate_limit > 0),
    CONSTRAINT chk_expires_at_future CHECK (
        expires_at IS NULL
        OR expires_at > created_at
    )
);

-- Create indexes
CREATE INDEX idx_merchant_api_keys_merchant_id ON merchant_api_keys (merchant_id);

CREATE INDEX idx_merchant_api_keys_prefix ON merchant_api_keys (prefix);

CREATE INDEX idx_merchant_api_keys_last_used ON merchant_api_keys (last_used_at);

CREATE INDEX idx_merchant_api_keys_expires ON merchant_api_keys (expires_at);

-- Create audit table for API key usage
CREATE TABLE merchant_api_key_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    api_key_id UUID NOT NULL REFERENCES merchant_api_keys (id),
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    user_agent VARCHAR(255),
    status_code INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for audit table
CREATE INDEX idx_merchant_api_key_usage_api_key ON merchant_api_key_usage (api_key_id);

CREATE INDEX idx_merchant_api_key_usage_created ON merchant_api_key_usage (created_at);

-- Create materialized view for API key analytics
CREATE MATERIALIZED VIEW mv_merchant_api_key_analytics AS
SELECT
    k.id as api_key_id,
    k.merchant_id,
    k.key_name,
    k.prefix,
    k.scopes,
    k.rate_limit,
    COUNT(u.id) as total_requests,
    COUNT(
        CASE
            WHEN u.status_code >= 200
            AND u.status_code < 300 THEN 1
        END
    ) as successful_requests,
    COUNT(
        CASE
            WHEN u.status_code >= 400 THEN 1
        END
    ) as failed_requests,
    COUNT(DISTINCT u.ip_address) as unique_ips,
    MIN(u.created_at) as first_used_at,
    MAX(u.created_at) as last_used_at,
    k.expires_at,
    CASE
        WHEN k.expires_at IS NOT NULL THEN CASE
            WHEN NOW() > k.expires_at THEN 'EXPIRED'
            ELSE CONCAT(
                EXTRACT(
                    DAY
                    FROM k.expires_at - NOW()
                )::integer,
                ' days left'
            )
        END
        ELSE 'NO EXPIRY'
    END as expiry_status
FROM
    merchant_api_keys k
    LEFT JOIN merchant_api_key_usage u ON k.id = u.api_key_id
WHERE
    k.deleted_at IS NULL
GROUP BY
    k.id,
    k.merchant_id,
    k.key_name,
    k.prefix,
    k.scopes,
    k.rate_limit,
    k.expires_at;

-- Create unique index for materialized view refresh
CREATE UNIQUE INDEX idx_mv_merchant_api_key_analytics ON mv_merchant_api_key_analytics (api_key_id);

-- Update refresh function to include new view
CREATE OR REPLACE FUNCTION refresh_all_materialized_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_transaction_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_account_balance_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_payment_method_performance;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_merchant_analytics;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_voucher_analytics;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_merchant_api_key_analytics;
END;
$$ LANGUAGE plpgsql;