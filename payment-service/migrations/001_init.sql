-- Payments table
CREATE TABLE IF NOT EXISTS payments (
    id             VARCHAR(36) PRIMARY KEY,
    order_id       VARCHAR(36) NOT NULL UNIQUE,
    transaction_id VARCHAR(255) NOT NULL,
    amount         BIGINT NOT NULL CHECK (amount > 0),
    status         VARCHAR(50) NOT NULL DEFAULT 'Authorized'
);
