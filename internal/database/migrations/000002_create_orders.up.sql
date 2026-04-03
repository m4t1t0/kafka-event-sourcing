CREATE TABLE IF NOT EXISTS orders (
    id             VARCHAR(36) PRIMARY KEY,
    client_id      VARCHAR(36) NOT NULL,
    client_name    VARCHAR(255) NOT NULL DEFAULT '',
    status         VARCHAR(20) NOT NULL DEFAULT 'placed',
    total          DECIMAL(12,2) NOT NULL DEFAULT 0,
    confirmed_by   VARCHAR(255) NOT NULL DEFAULT '',
    cancel_reason  VARCHAR(500) NOT NULL DEFAULT '',
    version        INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_orders_client_id ON orders (client_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders (status);

CREATE TABLE IF NOT EXISTS order_items (
    id         SERIAL PRIMARY KEY,
    order_id   VARCHAR(36) NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id VARCHAR(255) NOT NULL,
    name       VARCHAR(255) NOT NULL,
    quantity   INTEGER NOT NULL,
    price      DECIMAL(12,2) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items (order_id);
