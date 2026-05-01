CREATE TABLE IF NOT EXISTS payment_orders (
    id BIGSERIAL PRIMARY KEY,
    user_email VARCHAR(255) NOT NULL,
    user_name VARCHAR(100) NOT NULL,
    user_notes TEXT NULL,
    amount DECIMAL(20,2) NOT NULL,
    pay_amount DECIMAL(20,2) NOT NULL,
    fee_rate DECIMAL(10,4) NOT NULL DEFAULT 0,
    recharge_code VARCHAR(64) NOT NULL,
    out_trade_no VARCHAR(64) NOT NULL DEFAULT '',
    payment_type VARCHAR(30) NOT NULL,
    payment_trade_no VARCHAR(128) NOT NULL,
    pay_url TEXT NULL,
    qr_code TEXT NULL,
    qr_code_img TEXT NULL,
    order_type VARCHAR(20) NOT NULL DEFAULT 'balance',
    plan_id BIGINT NULL,
    subscription_group_id BIGINT NULL,
    subscription_days INTEGER NULL,
    provider_instance_id VARCHAR(64) NULL,
    provider_key VARCHAR(30) NULL,
    provider_snapshot JSONB NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    refund_amount DECIMAL(20,2) NOT NULL DEFAULT 0,
    refund_reason TEXT NULL,
    refund_at TIMESTAMPTZ NULL,
    force_refund BOOLEAN NOT NULL DEFAULT FALSE,
    refund_requested_at TIMESTAMPTZ NULL,
    refund_request_reason TEXT NULL,
    refund_requested_by VARCHAR(20) NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ NULL,
    completed_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,
    failed_reason TEXT NULL,
    client_ip VARCHAR(50) NOT NULL,
    src_host VARCHAR(255) NOT NULL,
    src_url TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id BIGINT NOT NULL REFERENCES users(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS paymentorder_out_trade_no
    ON payment_orders(out_trade_no)
    WHERE out_trade_no <> '';

CREATE INDEX IF NOT EXISTS paymentorder_user_id
    ON payment_orders(user_id);

CREATE INDEX IF NOT EXISTS paymentorder_status
    ON payment_orders(status);

CREATE INDEX IF NOT EXISTS paymentorder_expires_at
    ON payment_orders(expires_at);

CREATE INDEX IF NOT EXISTS paymentorder_created_at
    ON payment_orders(created_at);

CREATE INDEX IF NOT EXISTS paymentorder_paid_at
    ON payment_orders(paid_at);

CREATE INDEX IF NOT EXISTS paymentorder_payment_type_paid_at
    ON payment_orders(payment_type, paid_at);

CREATE INDEX IF NOT EXISTS paymentorder_order_type
    ON payment_orders(order_type);
