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
	DeveloperID  string
	Name         string
	LogoURL      string
	RedirectURLs []string
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

func generateAPIKey() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return "hell_yeah_" + hex.EncodeToString(bytes)
}

func (r *AppRepository) Create(ctx context.Context, developerID, name, logoURL string, redirectURLs []string) (*App, string, error) {
	apiKey := generateAPIKey()
	apiKeyHash := hashAPIKey(apiKey)

	app := &App{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO apps (developer_id, name, logo_url, redirect_urls, api_key_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, developer_id, name, logo_url, redirect_urls, api_key_hash, created_at, updated_at
	`, developerID, name, logoURL, redirectURLs, apiKeyHash).Scan(
		&app.ID, &app.DeveloperID, &app.Name, &app.LogoURL, &app.RedirectURLs, &app.APIKeyHash, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, "", err
	}
	return app, apiKey, nil
}

func (r *AppRepository) FindByID(ctx context.Context, id string) (*App, error) {
	app := &App{}
	err := r.db.QueryRow(ctx, `
		SELECT id, developer_id, name, logo_url, redirect_urls, api_key_hash, created_at, updated_at
		FROM apps WHERE id = $1
	`, id).Scan(
		&app.ID, &app.DeveloperID, &app.Name, &app.LogoURL, &app.RedirectURLs, &app.APIKeyHash, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (r *AppRepository) ListByDeveloper(ctx context.Context, developerID string) ([]*App, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, developer_id, name, logo_url, redirect_urls, api_key_hash, created_at, updated_at
		FROM apps WHERE developer_id = $1 ORDER BY created_at DESC
	`, developerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*App
	for rows.Next() {
		app := &App{}
		err := rows.Scan(&app.ID, &app.DeveloperID, &app.Name, &app.LogoURL, &app.RedirectURLs, &app.APIKeyHash, &app.CreatedAt, &app.UpdatedAt)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (r *AppRepository) Update(ctx context.Context, id string, name, logoURL *string, redirectURLs []string) (*App, error) {
	app := &App{}
	err := r.db.QueryRow(ctx, `
		UPDATE apps SET 
			name = COALESCE($2, name),
			logo_url = COALESCE($3, logo_url),
			redirect_urls = COALESCE($4, redirect_urls),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, developer_id, name, logo_url, redirect_urls, api_key_hash, created_at, updated_at
	`, id, name, logoURL, redirectURLs).Scan(
		&app.ID, &app.DeveloperID, &app.Name, &app.LogoURL, &app.RedirectURLs, &app.APIKeyHash, &app.CreatedAt, &app.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (r *AppRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM apps WHERE id = $1`, id)
	return err
}

func (r *AppRepository) RotateAPIKey(ctx context.Context, id string) (string, error) {
	apiKey := generateAPIKey()
	apiKeyHash := hashAPIKey(apiKey)

	_, err := r.db.Exec(ctx, `UPDATE apps SET api_key_hash = $2, updated_at = NOW() WHERE id = $1`, id, apiKeyHash)
	if err != nil {
		return "", err
	}
	return apiKey, nil
}

func hashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}
