-- Add up migration script here
CREATE TABLE voucher_redemptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    voucher_id UUID NOT NULL,
    account_id UUID NOT NULL,
    transaction_id UUID,
    redeemed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    CONSTRAINT fk_voucher FOREIGN KEY (voucher_id) REFERENCES vouchers (id),
    CONSTRAINT fk_account_redemption FOREIGN KEY (account_id) REFERENCES accounts (connect_id),
    CONSTRAINT fk_transaction_redemption FOREIGN KEY (transaction_id) REFERENCES transactions (id),
    UNIQUE (
        voucher_id,
        account_id,
        transaction_id
    )
);

CREATE INDEX idx_redemptions_account_id ON voucher_redemptions (account_id);

CREATE INDEX idx_redemptions_voucher_id ON voucher_redemptions (voucher_id);

CREATE INDEX idx_redemptions_transaction_id ON voucher_redemptions (transaction_id);