CREATE TABLE  IF NOT EXISTS  plans (
    id BIGSERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,              -- Free, Pro, Business
    storage_quota BIGINT NOT NULL,    -- all storage quota in bytes
    price BIGINT NOT NULL DEFAULT 0,  -- price
    discount_price BIGINT NOT NULL DEFAULT 0, -- discounted price
    duration_days INT NOT NULL,       -- 30, 365, 0 = lifetime
    description TEXT,
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO plans (
    name,
    storage_quota,
    price,
    discount_price,
    duration_days,
    description
) VALUES
('boss', 10 * 1024^3, 0, 0, 0, 'Pro plan for boss with 10 GB storage quota');