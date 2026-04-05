package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OAuthProvider struct {
	ID                    string
	AppID                 string
	Provider              string
	ClientID              string
	ClientSecretEncrypted string
	Scopes                []string
	Enabled               bool
	CreatedAt             time.Time
}

type OAuthProviderRepository struct {
	db *pgxpool.Pool
}

func NewOAuthProviderRepository(db *pgxpool.Pool) *OAuthProviderRepository {
	return &OAuthProviderRepository{db: db}
}

func (r *OAuthProviderRepository) Create(ctx context.Context, appID, provider, clientID, clientSecretEncrypted string, scopes []string) (*OAuthProvider, error) {
	op := &OAuthProvider{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO oauth_providers (app_id, provider, client_id, client_secret_encrypted, scopes) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, app_id, provider, client_id, client_secret_encrypted, scopes, enabled, created_at
	`, appID, provider, clientID, clientSecretEncrypted, scopes).Scan(
		&op.ID, &op.AppID, &op.Provider, &op.ClientID, &op.ClientSecretEncrypted, &op.Scopes, &op.Enabled, &op.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return op, nil
}

func (r *OAuthProviderRepository) FindByAppAndProvider(ctx context.Context, appID, provider string) (*OAuthProvider, error) {
	op := &OAuthProvider{}
	err := r.db.QueryRow(ctx, `
		SELECT id, app_id, provider, client_id, client_secret_encrypted, scopes, enabled, created_at
		FROM oauth_providers WHERE app_id = $1 AND provider = $2 AND enabled = true
	`, appID, provider).Scan(
		&op.ID, &op.AppID, &op.Provider, &op.ClientID, &op.ClientSecretEncrypted, &op.Scopes, &op.Enabled, &op.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return op, nil
}

func (r *OAuthProviderRepository) ListByAppID(ctx context.Context, appID string) ([]*OAuthProvider, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, app_id, provider, client_id, client_secret_encrypted, scopes, enabled, created_at
		FROM oauth_providers WHERE app_id = $1
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*OAuthProvider
	for rows.Next() {
		op := &OAuthProvider{}
		err := rows.Scan(&op.ID, &op.AppID, &op.Provider, &op.ClientID, &op.ClientSecretEncrypted, &op.Scopes, &op.Enabled, &op.CreatedAt)
		if err != nil {
			return nil, err
		}
		providers = append(providers, op)
	}
	return providers, nil
}

func (r *OAuthProviderRepository) Update(ctx context.Context, appID, provider string, clientID, clientSecretEncrypted *string, scopes []string, enabled *bool) (*OAuthProvider, error) {
	op := &OAuthProvider{}
	err := r.db.QueryRow(ctx, `
		UPDATE oauth_providers SET
			client_id = COALESCE($3, client_id),
			client_secret_encrypted = COALESCE($4, client_secret_encrypted),
			scopes = COALESCE($5, scopes),
			enabled = COALESCE($6, enabled)
		WHERE app_id = $1 AND provider = $2
		RETURNING id, app_id, provider, client_id, client_secret_encrypted, scopes, enabled, created_at
	`, appID, provider, clientID, clientSecretEncrypted, scopes, enabled).Scan(
		&op.ID, &op.AppID, &op.Provider, &op.ClientID, &op.ClientSecretEncrypted, &op.Scopes, &op.Enabled, &op.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return op, nil
}

func (r *OAuthProviderRepository) Delete(ctx context.Context, appID, provider string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM oauth_providers WHERE app_id = $1 AND provider = $2`, appID, provider)
	return err
}
