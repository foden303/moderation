-- name: CreateSubscription :one
INSERT INTO subscriptions (
  user_id, 
  plan_id, 
  started_at, 
  expired_at, 
  status,
  created_by
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetSubscriptionByID :one
SELECT * FROM subscriptions
WHERE id = $1 LIMIT 1;

-- name: ExpireSubscription :exec
UPDATE subscriptions
SET status = $3, expire_at = NOW()
WHERE user_id = $1 AND status = $2;

-- name: ExpireSubscriptions :exec
UPDATE subscriptions
SET status = $1, expire_at = NOW()
WHERE 
    status = $2
    AND expire_at IS NOT NULL
    AND expire_at <= NOW();

-- name: AutoResetQuotaDefault :exec
UPDATE users u
SET effective_storage_quota = $1,
    updated_at = NOW()
WHERE u.effective_storage_quota <> $1
AND EXISTS (
  SELECT 1 FROM subscriptions s
  WHERE s.user_id = u.id
    AND s.status = $2
);