-- name: CreateUser :exec
INSERT INTO users (id, mail_hash, stripe_customer_id) VALUES (?, ?, ?);

-- name: GetUser :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: GetUserByMailHash :one
SELECT * FROM users WHERE mail_hash = ? LIMIT 1;

-- name: UpdateUserStripeCustomerId :exec
UPDATE users SET stripe_customer_id = ? WHERE id = ?;
