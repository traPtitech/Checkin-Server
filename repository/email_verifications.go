package repository

import (
	"context"
	"database/sql"
	"time"
)

type EmailVerification struct {
	TokenHash    string
	Email        string
	RedirectPath string
	ExpiresAt    time.Time
	UsedAt       sql.NullTime
	CreatedAt    time.Time
}

type CreateEmailVerificationParams struct {
	TokenHash    string
	Email        string
	RedirectPath string
	ExpiresAt    time.Time
}

const createEmailVerification = `-- name: CreateEmailVerification :exec
INSERT INTO email_verifications (token_hash, email, redirect_path, expires_at) VALUES (?, ?, ?, ?)
`

const deleteUnusedEmailVerificationsByEmail = `-- name: DeleteUnusedEmailVerificationsByEmail :exec
DELETE FROM email_verifications WHERE email = ? AND used_at IS NULL
`

const getEmailVerificationByTokenHash = `-- name: GetEmailVerificationByTokenHash :one
SELECT token_hash, email, redirect_path, expires_at, used_at, created_at
FROM email_verifications
WHERE token_hash = ?
LIMIT 1
`

const markEmailVerificationUsed = `-- name: MarkEmailVerificationUsed :execrows
UPDATE email_verifications
SET used_at = ?
WHERE token_hash = ? AND used_at IS NULL
`

func (q *Queries) CreateEmailVerification(ctx context.Context, arg CreateEmailVerificationParams) error {
	_, err := q.db.ExecContext(ctx, createEmailVerification, arg.TokenHash, arg.Email, arg.RedirectPath, arg.ExpiresAt)
	return err
}

func (q *Queries) DeleteUnusedEmailVerificationsByEmail(ctx context.Context, email string) error {
	_, err := q.db.ExecContext(ctx, deleteUnusedEmailVerificationsByEmail, email)
	return err
}

func (q *Queries) GetEmailVerificationByTokenHash(ctx context.Context, tokenHash string) (EmailVerification, error) {
	row := q.db.QueryRowContext(ctx, getEmailVerificationByTokenHash, tokenHash)
	var item EmailVerification
	err := row.Scan(
		&item.TokenHash,
		&item.Email,
		&item.RedirectPath,
		&item.ExpiresAt,
		&item.UsedAt,
		&item.CreatedAt,
	)
	return item, err
}

func (q *Queries) MarkEmailVerificationUsed(ctx context.Context, tokenHash string, usedAt time.Time) (int64, error) {
	result, err := q.db.ExecContext(ctx, markEmailVerificationUsed, usedAt, tokenHash)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
