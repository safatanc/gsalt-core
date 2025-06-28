-- Add up migration script here
CREATE TYPE transaction_type AS ENUM (
  'TOPUP', 
  'TRANSFER_IN', 
  'TRANSFER_OUT', 
  'PAYMENT', 
  'GIFT_IN', 
  'GIFT_OUT', 
  'VOUCHER_REDEMPTION'
);

CREATE TYPE transaction_status AS ENUM (
  'PENDING',
  'COMPLETED',
  'FAILED',
  'CANCELLED'
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    account_id UUID NOT NULL,
    type transaction_type NOT NULL,
    amount DECIMAL(18, 2) NOT NULL,
    currency VARCHAR(5) NOT NULL DEFAULT 'IDR',
    status transaction_status NOT NULL DEFAULT 'PENDING',
    description TEXT,
    related_transaction_id UUID,
    source_account_id UUID,
    destination_account_id UUID,
    voucher_code VARCHAR(50),
    external_reference_id VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
    CONSTRAINT fk_account FOREIGN KEY (account_id) REFERENCES accounts (connect_id),
    CONSTRAINT fk_related_transaction FOREIGN KEY (related_transaction_id) REFERENCES transactions (id),
    CONSTRAINT fk_source_account FOREIGN KEY (source_account_id) REFERENCES accounts (connect_id),
    CONSTRAINT fk_destination_account FOREIGN KEY (destination_account_id) REFERENCES accounts (connect_id)
);

CREATE INDEX idx_transactions_account_id ON transactions (account_id);

CREATE INDEX idx_transactions_type ON transactions (type);

CREATE INDEX idx_transactions_status ON transactions (status);

CREATE INDEX idx_transactions_created_at ON transactions (created_at);

CREATE INDEX idx_transactions_external_reference_id ON transactions (external_reference_id);