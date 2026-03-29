package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Identity struct {
	ID             string
	UserID         string
	Provider       string
	ProviderUserID *string
	PasswordHash   *string
	CreatedAt      time.Time
}

type IdentityRepository struct {
	db *pgxpool.Pool
}

func NewIdentityRepository(db *pgxpool.Pool) *IdentityRepository {
	return &IdentityRepository{db: db}
}

func (r *IdentityRepository) Create(ctx context.Context, userID, provider string, providerUserID, passwordHash *string) (*Identity, error) {
	identity := &Identity{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO user_identities (user_id, provider, provider_user_id, password_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, provider, provider_user_id, password_hash, created_at
	`, userID, provider, providerUserID, passwordHash).Scan(
		&identity.ID, &identity.UserID, &identity.Provider,
		&identity.ProviderUserID, &identity.PasswordHash, &identity.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

func (r *IdentityRepository) FindByProviderID(ctx context.Context, provider, providerUserID string) (*Identity, error) {
	identity := &Identity{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_user_id, password_hash, created_at
		FROM user_identities WHERE provider = $1 AND provider_user_id = $2
	`, provider, providerUserID).Scan(
		&identity.ID, &identity.UserID, &identity.Provider,
		&identity.ProviderUserID, &identity.PasswordHash, &identity.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

func (r *IdentityRepository) FindByUserAndProvider(ctx context.Context, userID, provider string) (*Identity, error) {
	identity := &Identity{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, provider, provider_user_id, password_hash, created_at
		FROM user_identities WHERE user_id = $1 AND provider = $2
	`, userID, provider).Scan(
		&identity.ID, &identity.UserID, &identity.Provider,
		&identity.ProviderUserID, &identity.PasswordHash, &identity.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

func (r *IdentityRepository) UpdatePasswordHash(ctx context.Context, userID, newHash string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE user_identities SET password_hash = $2
		WHERE user_id = $1 AND provider = 'email'
	`, userID, newHash)
	return err
}
