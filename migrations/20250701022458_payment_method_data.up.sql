-- Add up migration script here

-- This script seeds the `payment_methods` table with initial data for the FLIP payment provider.
-- It is designed to be idempotent using `ON CONFLICT DO NOTHING`.

INSERT INTO
    payment_methods (
        name,
        code,
        currency,
        method_type,
        provider_code,
        provider_method_code,
        provider_method_type,
        payment_fee_flat,
        payment_fee_percent,
        is_available_for_topup,
        is_available_for_withdrawal
    )
VALUES
    -- Virtual Accounts (Flat Fee)
    (
        'Mandiri Virtual Account',
        'VA_MANDIRI',
        'IDR',
        'VIRTUAL_ACCOUNT',
        'FLIP',
        'mandiri',
        'virtual_account',
        4000,
        0,
        TRUE,
        FALSE
    ),
    (
        'Danamon Virtual Account',
        'VA_DANAMON',
        'IDR',
        'VIRTUAL_ACCOUNT',
        'FLIP',
        'danamon',
        'virtual_account',
        4000,
        0,
        TRUE,
        FALSE
    ),
    (
        'CIMB Virtual Account',
        'VA_CIMB',
        'IDR',
        'VIRTUAL_ACCOUNT',
        'FLIP',
        'cimb',
        'virtual_account',
        4000,
        0,
        TRUE,
        FALSE
    ),
    (
        'BNI Virtual Account',
        'VA_BNI',
        'IDR',
        'VIRTUAL_ACCOUNT',
        'FLIP',
        'bni',
        'virtual_account',
        4000,
        0,
        TRUE,
        FALSE
    ),
    (
        'BRI Virtual Account',
        'VA_BRI',
        'IDR',
        'VIRTUAL_ACCOUNT',
        'FLIP',
        'bri',
        'virtual_account',
        4000,
        0,
        TRUE,
        FALSE
    ),
    (
        'BCA Virtual Account',
        'VA_BCA',
        'IDR',
        'VIRTUAL_ACCOUNT',
        'FLIP',
        'bca',
        'virtual_account',
        4000,
        0,
        TRUE,
        FALSE
    ),
    (
        'Permata Virtual Account',
        'VA_PERMATA',
        'IDR',
        'VIRTUAL_ACCOUNT',
        'FLIP',
        'permata',
        'virtual_account',
        4000,
        0,
        TRUE,
        FALSE
    ),
    (
        'BSI Virtual Account',
        'VA_BSI',
        'IDR',
        'VIRTUAL_ACCOUNT',
        'FLIP',
        'bsm',
        'virtual_account',
        4000,
        0,
        TRUE,
        FALSE
    ),
    (
        'Seabank Virtual Account',
        'VA_SEABANK',
        'IDR',
        'VIRTUAL_ACCOUNT',
        'FLIP',
        'seabank',
        'virtual_account',
        4000,
        0,
        TRUE,
        FALSE
    ),

-- E-Wallets (Percentage Fee)
(
    'QRIS',
    'QRIS',
    'IDR',
    'EWALLET',
    'FLIP',
    'qris',
    'wallet_account',
    0,
    0.007,
    TRUE,
    FALSE
), -- 0.7%
(
    'OVO',
    'EWALLET_OVO',
    'IDR',
    'EWALLET',
    'FLIP',
    'ovo',
    'wallet_account',
    0,
    0.015,
    TRUE,
    FALSE
), -- 1.5%
(
    'ShopeePay',
    'EWALLET_SHOPEE',
    'IDR',
    'EWALLET',
    'FLIP',
    'shopeepay_app',
    'wallet_account',
    0,
    0.015,
    TRUE,
    FALSE
), -- 1.5%
(
    'LinkAja',
    'EWALLET_LINKAJA',
    'IDR',
    'EWALLET',
    'FLIP',
    'linkaja',
    'wallet_account',
    0,
    0.015,
    TRUE,
    FALSE
), -- 1.5%
(
    'DANA',
    'EWALLET_DANA',
    'IDR',
    'EWALLET',
    'FLIP',
    'dana',
    'wallet_account',
    0,
    0.015,
    TRUE,
    FALSE
), -- 1.5%

-- Retail Outlets (Flat Fee)
(
    'Alfamart',
    'RETAIL_ALFAMART',
    'IDR',
    'RETAIL_OUTLET',
    'FLIP',
    'alfamart',
    'online_to_offline_account',
    5000,
    0,
    TRUE,
    FALSE
),

-- Card Payments (Percentage Fee)
(
    'Credit/Debit Card',
    'CREDIT_CARD',
    'IDR',
    'CARD',
    'FLIP',
    'credit_card',
    'credit_card_account',
    0,
    0.025,
    TRUE,
    FALSE
) -- 2.5%
ON CONFLICT (code) DO NOTHING;