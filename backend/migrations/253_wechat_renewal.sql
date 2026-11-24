CREATE TABLE IF NOT EXISTS wechat_renewal_contracts (
    initial_order_id BIGINT PRIMARY KEY REFERENCES payment_orders(id),
    user_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    template_id TEXT NOT NULL,
    contract_code TEXT NOT NULL UNIQUE,
    contract_id TEXT UNIQUE,
    signed_payer_hash TEXT,
    paid_payer_hash TEXT,
    state TEXT NOT NULL CHECK (state IN ('pending', 'active', 'cancelled', 'failed')),
    period_end TIMESTAMPTZ,
    failure_code TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS wechat_renewal_open_contract_idx
    ON wechat_renewal_contracts(user_id, group_id) WHERE state IN ('pending', 'active');
CREATE TABLE IF NOT EXISTS wechat_renewal_cycles (
    order_id BIGINT PRIMARY KEY REFERENCES payment_orders(id),
    initial_order_id BIGINT NOT NULL REFERENCES wechat_renewal_contracts(initial_order_id),
    period_end TIMESTAMPTZ NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('queued', 'processing', 'completed', 'failed')),
    next_check_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (initial_order_id, period_end)
);
CREATE INDEX IF NOT EXISTS wechat_renewal_cycle_due_idx ON wechat_renewal_cycles(next_check_at)
    WHERE state IN ('queued', 'processing');
