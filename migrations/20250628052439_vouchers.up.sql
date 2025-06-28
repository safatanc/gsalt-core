-- Add up migration script here
CREATE TYPE voucher_type AS ENUM (
  'BALANCE', 'LOYALTY_POINTS', 'DISCOUNT'
);

CREATE TYPE voucher_status AS ENUM (
  'ACTIVE', 'INACTIVE', 'REDEEMED', 'EXPIRED'
);

CREATE TABLE vouchers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type voucher_type NOT NULL,
    value DECIMAL(18, 2) NOT NULL,
    currency VARCHAR(5) NOT NULL DEFAULT 'IDR',
    loyalty_points_value BIGINT,
    discount_percentage DECIMAL(5, 2),
    discount_amount DECIMAL(18, 2),
    max_redeem_count INT DEFAULT 1,
    current_redeem_count INT DEFAULT 0,
    valid_from TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP WITH TIME ZONE,
    status voucher_status NOT NULL DEFAULT 'ACTIVE',
    created_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE INDEX idx_vouchers_code ON vouchers (code);

CREATE INDEX idx_vouchers_status ON vouchers (status);

CREATE INDEX idx_vouchers_valid_until ON vouchers (valid_until);