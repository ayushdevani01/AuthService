package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ayushdevan01/AuthService/services/user-service/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidState    = errors.New("invalid or expired OAuth state")
	ErrProviderNotConf = errors.New("OAuth provider not configured for this app")
	ErrOAuthFailed     = errors.New("OAuth token exchange failed")
)

// OAuthProviderConfig from the oauth_providers table
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// OAuth state stored in Redis
type oauthState struct {
	AppID       string `json:"app_id"`
	Provider    string `json:"provider"`
	RedirectURI string `json:"redirect_uri"`
}

// Google user info response
type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

// GitHub user info response
type githubUserInfo struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHub email response
type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

type OAuthService struct {
	redisClient   *redis.Client
	db            *pgxpool.Pool
	userService   *UserService
	encryptionKey string
	platformURL   string
}

func NewOAuthService(redisClient *redis.Client, db *pgxpool.Pool, userService *UserService, encryptionKey, platformURL string) *OAuthService {
	return &OAuthService{
		redisClient:   redisClient,
		db:            db,
		userService:   userService,
		encryptionKey: encryptionKey,
		platformURL:   platformURL,
	}
}

func (s *OAuthService) getProviderConfig(ctx context.Context, appID, provider string) (*OAuthProviderConfig, error) {
	var clientID, clientSecretEncrypted string
	var scopes []string

	err := s.db.QueryRow(ctx, `
		SELECT client_id, client_secret_encrypted, scopes
		FROM oauth_providers
		WHERE app_id = (SELECT id FROM apps WHERE app_id = $1) AND provider = $2 AND enabled = true
	`, appID, provider).Scan(&clientID, &clientSecretEncrypted, &scopes)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProviderNotConf
		}
		return nil, err
	}

	clientSecret, err := decryptAES(clientSecretEncrypted, s.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt client secret: %w", err)
	}

	return &OAuthProviderConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       scopes,
	}, nil
}

func (s *OAuthService) InitiateOAuth(ctx context.Context, appID, provider, redirectURI string) (string, string, error) {
	// Verify provider is configured
	providerConfig, err := s.getProviderConfig(ctx, appID, provider)
	if err != nil {
		return "", "", err
	}

	// Generate state
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	// Store state in Redis with 10 min TTL
	stateData, _ := json.Marshal(oauthState{
		AppID:       appID,
		Provider:    provider,
		RedirectURI: redirectURI,
	})
	s.redisClient.Set(ctx, fmt.Sprintf("oauth_state:%s", state), stateData, 10*time.Minute)

	// Build authorization URL
	authURL := s.buildAuthorizationURL(provider, providerConfig.ClientID, state, providerConfig.Scopes)

	return authURL, state, nil
}

func (s *OAuthService) HandleOAuthCallback(ctx context.Context, provider, code, state string) (*repository.User, string, string, bool, error) {
	// Validate state from Redis
	redisKey := fmt.Sprintf("oauth_state:%s", state)
	stateJSON, err := s.redisClient.Get(ctx, redisKey).Result()
	if err != nil {
		return nil, "", "", false, ErrInvalidState
	}
	s.redisClient.Del(ctx, redisKey)

	var stateData oauthState
	if err := json.Unmarshal([]byte(stateJSON), &stateData); err != nil {
		return nil, "", "", false, ErrInvalidState
	}

	// Get provider config
	providerConfig, err := s.getProviderConfig(ctx, stateData.AppID, provider)
	if err != nil {
		return nil, "", "", false, err
	}

	// Exchange code for tokens and get user info
	var email, name, avatarURL, providerUserID string
	var emailVerified bool

	switch provider {
	case "google":
		email, name, avatarURL, providerUserID, emailVerified, err = s.handleGoogleCallback(ctx, code, providerConfig)
	case "github":
		email, name, avatarURL, providerUserID, emailVerified, err = s.handleGithubCallback(ctx, code, providerConfig)
	default:
		return nil, "", "", false, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, "", "", false, err
	}

	// Check if user is new
	_, findErr := s.userService.GetUserByEmail(ctx, stateData.AppID, email)
	isNewUser := errors.Is(findErr, ErrUserNotFound)

	// Create or update user
	providerUserIDPtr := &providerUserID
	user, err := s.userService.CreateUser(ctx, stateData.AppID, email, name, avatarURL, provider, providerUserIDPtr, nil, emailVerified)
	if err != nil {
		return nil, "", "", false, err
	}

	return user, stateData.AppID, stateData.RedirectURI, isNewUser, nil
}

func (s *OAuthService) handleGoogleCallback(ctx context.Context, code string, config *OAuthProviderConfig) (string, string, string, string, bool, error) {
	callbackURL := s.platformURL + "/oauth/callback/google"
	tokenResp, err := exchangeCodeForToken("https://oauth2.googleapis.com/token", code, config.ClientID, config.ClientSecret, callbackURL)
	if err != nil {
		return "", "", "", "", false, err
	}

	accessToken, ok := tokenResp["access_token"].(string)
	if !ok {
		return "", "", "", "", false, ErrOAuthFailed
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", "", false, fmt.Errorf("failed to fetch Google user info: %w", err)
	}
	defer resp.Body.Close()

	var userInfo googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", "", "", "", false, fmt.Errorf("failed to decode Google user info: %w", err)
	}

	return userInfo.Email, userInfo.Name, userInfo.Picture, userInfo.ID, userInfo.VerifiedEmail, nil
}

func (s *OAuthService) handleGithubCallback(ctx context.Context, code string, config *OAuthProviderConfig) (string, string, string, string, bool, error) {
	callbackURL := s.platformURL + "/oauth/callback/github"
	tokenResp, err := exchangeCodeForTokenJSON("https://github.com/login/oauth/access_token", code, config.ClientID, config.ClientSecret, callbackURL)
	if err != nil {
		return "", "", "", "", false, err
	}

	accessToken, ok := tokenResp["access_token"].(string)
	if !ok {
		return "", "", "", "", false, ErrOAuthFailed
	}

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", "", false, fmt.Errorf("failed to fetch GitHub user info: %w", err)
	}
	defer resp.Body.Close()

	var userInfo githubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return "", "", "", "", false, fmt.Errorf("failed to decode GitHub user info: %w", err)
	}

	email := userInfo.Email
	emailVerified := false

	if email == "" {
		email, emailVerified, err = s.fetchGithubEmail(ctx, accessToken)
		if err != nil {
			return "", "", "", "", false, err
		}
	} else {
		emailVerified = true
	}

	providerUserID := fmt.Sprintf("%d", userInfo.ID)
	return email, userInfo.Name, userInfo.AvatarURL, providerUserID, emailVerified, nil
}

func (s *OAuthService) fetchGithubEmail(ctx context.Context, accessToken string) (string, bool, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()

	var emails []githubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", false, err
	}

	for _, e := range emails {
		if e.Primary {
			return e.Email, e.Verified, nil
		}
	}
	if len(emails) > 0 {
		return emails[0].Email, emails[0].Verified, nil
	}
	return "", false, fmt.Errorf("no email found from GitHub")
}

// exchangeCodeForToken exchanges an authorization code using form POST
func exchangeCodeForToken(tokenURL, code, clientID, clientSecret, redirectURI string) (map[string]interface{}, error) {
	resp, err := http.PostForm(tokenURL, map[string][]string{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"redirect_uri":  {redirectURI},
	})
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if errMsg, ok := result["error"]; ok {
		return nil, fmt.Errorf("OAuth error: %v", errMsg)
	}

	return result, nil
}

// exchangeCodeForTokenJSON is for GitHub which needs Accept: application/json header
func exchangeCodeForTokenJSON(tokenURL, code, clientID, clientSecret, redirectURI string) (map[string]interface{}, error) {
	req, _ := http.NewRequest("POST", tokenURL, nil)
	q := req.URL.Query()
	q.Set("client_id", clientID)
	q.Set("client_secret", clientSecret)
	q.Set("code", code)
	q.Set("redirect_uri", redirectURI)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	if errMsg, ok := result["error"]; ok {
		return nil, fmt.Errorf("OAuth error: %v", errMsg)
	}

	return result, nil
}

// buildAuthorizationURL builds provider-specific authorization URL
func (s *OAuthService) buildAuthorizationURL(provider, clientID, state string, scopes []string) string {
	callbackURL := fmt.Sprintf("%s/oauth/callback/%s", s.platformURL, provider)

	switch provider {
	case "google":
		scopeStr := "openid email profile"
		if len(scopes) > 0 {
			scopeStr = joinScopes(scopes)
		}
		return fmt.Sprintf(
			"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s&access_type=offline",
			clientID, callbackURL, scopeStr, state,
		)
	case "github":
		scopeStr := "user:email"
		if len(scopes) > 0 {
			scopeStr = joinScopes(scopes)
		}
		return fmt.Sprintf(
			"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
			clientID, callbackURL, scopeStr, state,
		)
	default:
		return ""
	}
}

func joinScopes(scopes []string) string {
	result := ""
	for i, s := range scopes {
		if i > 0 {
			result += " "
		}
		result += s
	}
	return result
}

// decryptAES decrypts AES-256-GCM encrypted text (same as developer-service/auth/encryption.go)
func decryptAES(ciphertextB64, key string) (string, error) {
	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return "", errors.New("encryption key must be 32 bytes")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("invalid ciphertext")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
