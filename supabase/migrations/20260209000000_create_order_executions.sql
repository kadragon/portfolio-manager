-- Create order_executions table for rebalance order history
CREATE TABLE IF NOT EXISTS order_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticker TEXT NOT NULL,
    side TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    currency TEXT NOT NULL,
    exchange TEXT,
    status TEXT NOT NULL,
    message TEXT NOT NULL DEFAULT '',
    raw_response JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

ALTER TABLE order_executions ENABLE ROW LEVEL SECURITY;

CREATE INDEX IF NOT EXISTS idx_order_executions_created_at
    ON order_executions(created_at DESC);
