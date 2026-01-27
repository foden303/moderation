CREATE TABLE subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    started_at TIMESTAMPTZ NOT NULL,
    expired_at TIMESTAMPTZ,
    status TEXT NOT NULL CHECK (status IN ('active','expired','canceled')),
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ
      NOT NULL DEFAULT now()
);

-- Ensure a user can have only one active subscription at a time
CREATE UNIQUE INDEX uniq_active_sub
ON subscriptions(user_id)
WHERE status = 'active';