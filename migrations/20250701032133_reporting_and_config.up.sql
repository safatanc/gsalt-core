-- Add up migration script here

-- Create system configuration table
CREATE TABLE system_configurations (
    id uuid DEFAULT gen_random_uuid () NOT NULL,
    category varchar(50) NOT NULL,
    key varchar(100) NOT NULL,
    value jsonb NOT NULL,
    description text,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_system_configurations PRIMARY KEY (id),
    CONSTRAINT uq_system_configurations_category_key UNIQUE (category, key)
);

-- Create index for faster config lookups
CREATE INDEX idx_system_configurations_category_key ON system_configurations (category, key)
WHERE
    is_active = true;

-- Insert default configurations
INSERT INTO
    system_configurations (
        category,
        key,
        value,
        description
    )
VALUES (
        'transaction_limits',
        'default_limits',
        jsonb_build_object(
            'min_topup_gsalt',
            1000,
            'max_topup_gsalt',
            5000000,
            'min_transfer_gsalt',
            100,
            'max_transfer_gsalt',
            2500000,
            'min_payment_gsalt',
            100,
            'max_payment_gsalt',
            1000000,
            'daily_transfer_limit',
            10000000,
            'daily_payment_limit',
            5000000
        ),
        'Default transaction limits in GSALT units'
    ),
    (
        'exchange_rates',
        'gsalt_rates',
        jsonb_build_object(
            'IDR',
            1000,
            'USD',
            15000,
            'SGD',
            11000
        ),
        'Exchange rates for 1 GSALT'
    ),
    (
        'fees',
        'transaction_fees',
        jsonb_build_object(
            'topup_percentage',
            0.5,
            'transfer_percentage',
            0.1,
            'withdrawal_percentage',
            0.5,
            'min_fee_gsalt',
            100
        ),
        'Transaction fees in percentage and minimum fee in GSALT units'
    );

-- Create materialized view for daily transaction summary
CREATE MATERIALIZED VIEW mv_daily_transaction_summary AS
SELECT
    date_trunc('day', created_at) AS transaction_date,
    type,
    status,
    COUNT(*) as transaction_count,
    SUM(amount_gsalt_units) as total_amount_gsalt_units,
    SUM(fee_gsalt_units) as total_fee_gsalt_units,
    COUNT(
        CASE
            WHEN status = 'COMPLETED' THEN 1
        END
    ) as completed_count,
    COUNT(
        CASE
            WHEN status = 'FAILED' THEN 1
        END
    ) as failed_count
FROM transactions
GROUP BY
    date_trunc('day', created_at),
    type,
    status;

-- Create unique index for materialized view refresh
CREATE UNIQUE INDEX idx_mv_daily_transaction_summary 
    ON mv_daily_transaction_summary(transaction_date, type, status);

-- Create materialized view for account balance summary
CREATE MATERIALIZED VIEW mv_account_balance_summary AS
SELECT
    a.account_type,
    COUNT(*) as account_count,
    SUM(a.balance) as total_balance_gsalt_units,
    AVG(a.balance) as avg_balance_gsalt_units,
    MIN(a.balance) as min_balance_gsalt_units,
    MAX(a.balance) as max_balance_gsalt_units,
    COUNT(
        CASE
            WHEN a.balance > 0 THEN 1
        END
    ) as active_accounts,
    COUNT(
        CASE
            WHEN a.kyc_status = 'VERIFIED' THEN 1
        END
    ) as verified_accounts
FROM accounts a
WHERE
    a.deleted_at IS NULL
GROUP BY
    a.account_type;

-- Create unique index for materialized view refresh
CREATE UNIQUE INDEX idx_mv_account_balance_summary ON mv_account_balance_summary (account_type);

-- Create function to refresh all materialized views
CREATE OR REPLACE FUNCTION refresh_all_materialized_views()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_daily_transaction_summary;
    REFRESH MATERIALIZED VIEW CONCURRENTLY mv_account_balance_summary;
END;
$$ LANGUAGE plpgsql;