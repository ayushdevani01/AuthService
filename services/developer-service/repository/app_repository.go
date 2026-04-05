package repository

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	ID           string
	AppID        string
	DeveloperID  string
	Name         string
	LogoURL      string
	RedirectURLs []string
	RequireEmailVerification bool
	APIKeyHash   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AppRepository struct {
	db *pgxpool.Pool
}

func NewAppRepository(db *pgxpool.Pool) *AppRepository {
	return &AppRepository{db: db}
}

func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return "hell_yeah_" + hex.EncodeToString(bytes), nil
}

func generateAppID() (string, error) {
	bytes := make([]byte, 12)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return "app_" + hex.EncodeToString(bytes), nil
}

func (r *AppRepository) Create(ctx context.Context, developerID, name, logoURL string, redirectURLs []string, requireEmailVerification bool) (*App, string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, "", err
	}
	appID, err := generateAppID()
	if err != nil {
		return nil, "", err
	}
	apiKeyHash := hashAPIKey(apiKey)

	app := &App{}
	err = r.db.QueryRow(ctx, `
		INSERT INTO apps (developer_id, name, app_id, logo_url, redirect_urls, require_email_verification, api_key_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, app_id, developer_id, name, logo_url, redirect_urls, require_email_verification, api_key_hash, created_at, updated_at
	`, developerID, name, appID, logoURL, redirectURLs, requireEmailVerification, apiKeyHash).Scan(
		&app.ID, &app.AppID, &app.DeveloperID, &app.Name, &app.LogoURL, &app.RedirectURLs, &app.RequireEmailVerification, &app.APIKeyHash, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, "", err
	}
	return app, apiKey, nil
}

func (r *AppRepository) FindByID(ctx context.Context, id string) (*App, error) {
	app := &App{}
	err := r.db.QueryRow(ctx, `
		SELECT id, app_id, developer_id, name, logo_url, redirect_urls, require_email_verification, api_key_hash, created_at, updated_at
		FROM apps WHERE id::text = $1 OR app_id = $1
	`, id).Scan(
		&app.ID, &app.AppID, &app.DeveloperID, &app.Name, &app.LogoURL, &app.RedirectURLs, &app.RequireEmailVerification, &app.APIKeyHash, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (r *AppRepository) ListByDeveloper(ctx context.Context, developerID string) ([]*App, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, app_id, developer_id, name, logo_url, redirect_urls, require_email_verification, api_key_hash, created_at, updated_at
		FROM apps WHERE developer_id = $1 ORDER BY created_at DESC
	`, developerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*App
	for rows.Next() {
		app := &App{}
		err := rows.Scan(&app.ID, &app.AppID, &app.DeveloperID, &app.Name, &app.LogoURL, &app.RedirectURLs, &app.RequireEmailVerification, &app.APIKeyHash, &app.CreatedAt, &app.UpdatedAt)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (r *AppRepository) Update(ctx context.Context, app *App) (*App, error) {
	updatedApp := &App{}
	err := r.db.QueryRow(ctx, `
		UPDATE apps SET 
			name = $2,
			logo_url = $3,
			redirect_urls = $4,
			require_email_verification = $5,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, app_id, developer_id, name, logo_url, redirect_urls, require_email_verification, api_key_hash, created_at, updated_at
	`, app.ID, app.Name, app.LogoURL, app.RedirectURLs, app.RequireEmailVerification).Scan(
		&updatedApp.ID, &updatedApp.AppID, &updatedApp.DeveloperID, &updatedApp.Name, &updatedApp.LogoURL, &updatedApp.RedirectURLs, &updatedApp.RequireEmailVerification, &updatedApp.APIKeyHash, &updatedApp.CreatedAt, &updatedApp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return updatedApp, nil
}

func (r *AppRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM apps WHERE id = $1`, id)
	return err
}

func (r *AppRepository) RotateAPIKey(ctx context.Context, id string) (string, error) {
	apiKey, err := generateAPIKey()
	if err != nil {
		return "", err
	}
	apiKeyHash := hashAPIKey(apiKey)

	_, err = r.db.Exec(ctx, `UPDATE apps SET api_key_hash = $2, updated_at = NOW() WHERE id = $1`, id, apiKeyHash)
	if err != nil {
		return "", err
	}
	return apiKey, nil
}

func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}
