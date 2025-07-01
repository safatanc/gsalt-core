-- Add up migration script here

-- Create audit_action ENUM type
CREATE TYPE audit_action AS ENUM (
    'CREATE',
    'UPDATE',
    'DELETE',
    'STATUS_CHANGE'
);

-- Create transaction_status_history table
CREATE TABLE transaction_status_history (
    id uuid DEFAULT gen_random_uuid () NOT NULL,
    transaction_id uuid NOT NULL,
    from_status transaction_status,
    to_status transaction_status NOT NULL,
    reason text,
    metadata jsonb,
    created_by uuid,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_transaction_status_history PRIMARY KEY (id),
    CONSTRAINT fk_transaction_status_history_transaction FOREIGN KEY (transaction_id) REFERENCES transactions (id)
);

-- Create audit_logs table for general auditing
CREATE TABLE audit_logs (
    id uuid DEFAULT gen_random_uuid () NOT NULL,
    table_name varchar(50) NOT NULL,
    record_id uuid NOT NULL,
    action audit_action NOT NULL,
    old_data jsonb,
    new_data jsonb,
    changed_by uuid,
    changed_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_audit_logs PRIMARY KEY (id)
);

-- Create indexes
CREATE INDEX idx_transaction_status_history_transaction_id ON transaction_status_history (transaction_id);

CREATE INDEX idx_transaction_status_history_created_at ON transaction_status_history (created_at);

CREATE INDEX idx_audit_logs_table_record ON audit_logs (table_name, record_id);

CREATE INDEX idx_audit_logs_changed_at ON audit_logs (changed_at);

CREATE INDEX idx_audit_logs_action ON audit_logs (action);

-- Add trigger function to automatically log transaction status changes
CREATE OR REPLACE FUNCTION log_transaction_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'UPDATE' AND OLD.status IS DISTINCT FROM NEW.status) THEN
        INSERT INTO transaction_status_history (
            transaction_id,
            from_status,
            to_status,
            reason,
            metadata
        ) VALUES (
            NEW.id,
            OLD.status,
            NEW.status,
            CASE 
                WHEN NEW.status = 'COMPLETED' THEN 'Transaction completed successfully'
                WHEN NEW.status = 'FAILED' THEN 'Transaction failed'
                WHEN NEW.status = 'CANCELLED' THEN 'Transaction cancelled'
                ELSE 'Status changed'
            END,
            jsonb_build_object(
                'payment_method', NEW.payment_method,
                'external_payment_id', NEW.external_payment_id,
                'processing_status', NEW.processing_status
            )
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for transaction status changes
CREATE TRIGGER trg_transaction_status_change
    AFTER UPDATE ON transactions
    FOR EACH ROW
    EXECUTE FUNCTION log_transaction_status_change();