-- name: GetPlanByID :one
SELECT * FROM plans
WHERE id = $1 LIMIT 1;

-- name: CreatePlan :one
INSERT INTO plans (
  name, 
  storage_quota, 
  price, 
  discount_price, 
  duration_days,
  description
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetPlans :many
SELECT * FROM plans
ORDER BY created_at DESC;

-- name: GetPlansWithCursor :many
SELECT * FROM plans
WHERE (created_at, id) < ($1, $2)
ORDER BY created_at DESC, id DESC
LIMIT $3;

-- name: GetPlansFirstPage :many
SELECT * FROM plans
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: GetPlansWithOffset :many
SELECT * FROM plans
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: CountPlans :one
SELECT COUNT(id) FROM plans;