package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PasswordResetToken struct {
	ID        string
	UserID    string
	AppID     string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type PasswordResetRepository struct {
	db *pgxpool.Pool
}

func NewPasswordResetRepository(db *pgxpool.Pool) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func (r *PasswordResetRepository) Create(ctx context.Context, userID, appID, tokenHash string, expiresAt time.Time) (*PasswordResetToken, error) {
	t := &PasswordResetToken{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO password_reset_tokens (user_id, app_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, app_id, token_hash, expires_at, used_at, created_at
	`, userID, appID, tokenHash, expiresAt).Scan(
		&t.ID, &t.UserID, &t.AppID, &t.TokenHash,
		&t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *PasswordResetRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	t := &PasswordResetToken{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, app_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
	`, tokenHash).Scan(
		&t.ID, &t.UserID, &t.AppID, &t.TokenHash,
		&t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

func (r *PasswordResetRepository) FindByTokenHashAny(ctx context.Context, tokenHash string) (*PasswordResetToken, error) {
	t := &PasswordResetToken{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, app_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
	`, tokenHash).Scan(
		&t.ID, &t.UserID, &t.AppID, &t.TokenHash,
		&t.ExpiresAt, &t.UsedAt, &t.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

func (r *PasswordResetRepository) ConsumeValidToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx, `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}

	return userID, nil
}

func (r *PasswordResetRepository) MarkUsed(ctx context.Context, tokenID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1
	`, tokenID)
	return err
}
