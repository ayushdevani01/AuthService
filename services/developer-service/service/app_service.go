package service

import (
	"context"
	"errors"

	"github.com/ayushdevan01/AuthService/services/developer-service/auth"
	"github.com/ayushdevan01/AuthService/services/developer-service/repository"
	"github.com/jackc/pgx/v5"
)

var (
	ErrAppNotFound      = errors.New("app not found")
	ErrNotAppOwner      = errors.New("not the owner of this app")
	ErrProviderNotFound = errors.New("oauth provider not found")
)

type AppService struct {
	appRepo        *repository.AppRepository
	signingKeyRepo *repository.SigningKeyRepository
	oauthRepo      *repository.OAuthProviderRepository
	encryptionKey  string
}

func NewAppService(appRepo *repository.AppRepository, signingKeyRepo *repository.SigningKeyRepository, oauthRepo *repository.OAuthProviderRepository, encryptionKey string) *AppService {
	return &AppService{
		appRepo:        appRepo,
		signingKeyRepo: signingKeyRepo,
		oauthRepo:      oauthRepo,
		encryptionKey:  encryptionKey,
	}
}

type CreateAppResponse struct {
	App        *repository.App
	APIKey     string
	SigningKey *repository.SigningKey
}

func (s *AppService) CreateApp(ctx context.Context, developerID, name, logoURL string, redirectURLs []string, requireEmailVerification bool) (*CreateAppResponse, error) {
	app, apiKey, err := s.appRepo.Create(ctx, developerID, name, logoURL, redirectURLs, requireEmailVerification)
	if err != nil {
		return nil, err
	}

	signingKey, err := s.signingKeyRepo.Create(ctx, app.ID)
	if err != nil {
		return nil, err
	}

	return &CreateAppResponse{
		App:        app,
		APIKey:     apiKey,
		SigningKey: signingKey,
	}, nil
}

func (s *AppService) GetApp(ctx context.Context, appID string) (*repository.App, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}
	return app, nil
}

func (s *AppService) ListApps(ctx context.Context, developerID string) ([]*repository.App, error) {
	return s.appRepo.ListByDeveloper(ctx, developerID)
}

func (s *AppService) UpdateApp(ctx context.Context, appID, developerID string, name, logoURL *string, redirectURLs []string, requireEmailVerification *bool) (*repository.App, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	if app.DeveloperID != developerID {
		return nil, ErrNotAppOwner
	}

	if name != nil {
		app.Name = *name
	}
	if logoURL != nil {
		app.LogoURL = *logoURL
	}
	if redirectURLs != nil {
		app.RedirectURLs = redirectURLs
	}
	if requireEmailVerification != nil {
		app.RequireEmailVerification = *requireEmailVerification
	}

	return s.appRepo.Update(ctx, app)
}

func (s *AppService) DeleteApp(ctx context.Context, appID, developerID string) error {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAppNotFound
		}
		return err
	}

	if app.DeveloperID != developerID {
		return ErrNotAppOwner
	}

	return s.appRepo.Delete(ctx, app.ID)
}

func (s *AppService) VerifyAPIKey(ctx context.Context, appID, apiKey string) (bool, error) {
	return s.appRepo.VerifyAPIKey(ctx, appID, apiKey)
}

func (s *AppService) RotateAPIKey(ctx context.Context, appID, developerID string) (string, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAppNotFound
		}
		return "", err
	}

	if app.DeveloperID != developerID {
		return "", ErrNotAppOwner
	}

	return s.appRepo.RotateAPIKey(ctx, app.ID)
}

func (s *AppService) RotateSigningKeys(ctx context.Context, appID, developerID string, gracePeriodHours int) (*repository.SigningKey, *repository.SigningKey, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, ErrAppNotFound
		}
		return nil, nil, err
	}

	if app.DeveloperID != developerID {
		return nil, nil, ErrNotAppOwner
	}

	if gracePeriodHours <= 0 {
		gracePeriodHours = 24
	}

	return s.signingKeyRepo.Rotate(ctx, app.ID, gracePeriodHours)
}

func (s *AppService) ListSigningKeys(ctx context.Context, appID, developerID string, includeExpired bool) ([]*repository.SigningKey, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	if app.DeveloperID != developerID {
		return nil, ErrNotAppOwner
	}

	return s.signingKeyRepo.ListByAppID(ctx, app.ID, includeExpired)
}

type ActiveSigningKeyResponse struct {
	Key        *repository.SigningKey
	PrivateKey string // Decrypted private key
}

func (s *AppService) GetActiveSigningKey(ctx context.Context, appID string) (*ActiveSigningKeyResponse, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	key, err := s.signingKeyRepo.GetActiveByAppID(ctx, app.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	decryptedKey, err := auth.Decrypt(key.PrivateKeyEncrypted, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	return &ActiveSigningKeyResponse{
		Key:        key,
		PrivateKey: decryptedKey,
	}, nil
}

// OAuth Provider methods
func (s *AppService) AddOAuthProvider(ctx context.Context, appID, developerID, provider, clientID, clientSecret string, scopes []string) (*repository.OAuthProvider, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	if app.DeveloperID != developerID {
		return nil, ErrNotAppOwner
	}

	encryptedSecret, err := auth.Encrypt(clientSecret, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	return s.oauthRepo.Create(ctx, app.ID, provider, clientID, encryptedSecret, scopes)
}

type OAuthProviderResponse struct {
	Provider     *repository.OAuthProvider
	ClientSecret string // Decrypted client secret
}

func (s *AppService) GetOAuthProvider(ctx context.Context, appID, provider string) (*OAuthProviderResponse, error) {
	op, err := s.oauthRepo.FindByAppAndProvider(ctx, appID, provider)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProviderNotFound
		}
		return nil, err
	}

	decryptedSecret, err := auth.Decrypt(op.ClientSecretEncrypted, s.encryptionKey)
	if err != nil {
		return nil, err
	}

	return &OAuthProviderResponse{
		Provider:     op,
		ClientSecret: decryptedSecret,
	}, nil
}

func (s *AppService) ListOAuthProviders(ctx context.Context, appID, developerID string) ([]*repository.OAuthProvider, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	if app.DeveloperID != developerID {
		return nil, ErrNotAppOwner
	}

	return s.oauthRepo.ListByAppID(ctx, app.ID)
}

func (s *AppService) UpdateOAuthProvider(ctx context.Context, appID, developerID, provider string, clientID, clientSecret *string, scopes []string, enabled *bool) (*repository.OAuthProvider, error) {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAppNotFound
		}
		return nil, err
	}

	if app.DeveloperID != developerID {
		return nil, ErrNotAppOwner
	}

	var encryptedSecret *string
	if clientSecret != nil {
		encrypted, err := auth.Encrypt(*clientSecret, s.encryptionKey)
		if err != nil {
			return nil, err
		}
		encryptedSecret = &encrypted
	}

	return s.oauthRepo.Update(ctx, app.ID, provider, clientID, encryptedSecret, scopes, enabled)
}

func (s *AppService) DeleteOAuthProvider(ctx context.Context, appID, developerID, provider string) error {
	app, err := s.appRepo.FindByID(ctx, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAppNotFound
		}
		return err
	}

	if app.DeveloperID != developerID {
		return ErrNotAppOwner
	}

	return s.oauthRepo.Delete(ctx, app.ID, provider)
}
