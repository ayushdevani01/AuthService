package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ayushdevan01/AuthService/services/user-service/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidState     = errors.New("invalid or expired OAuth state")
	ErrProviderNotConf  = errors.New("OAuth provider not configured for this app")
	ErrOAuthFailed      = errors.New("OAuth token exchange failed")
	ErrInvalidVerifier  = errors.New("invalid PKCE code verifier")
	ErrProviderMismatch = errors.New("oauth_provider_mismatch")
)

// OAuthProviderConfig from the oauth_providers table
type OAuthProviderConfig struct {
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// OAuth state stored in Redis
type oauthState struct {
	AppID               string `json:"app_id"`
	PublicAppID         string `json:"public_app_id,omitempty"`
	Provider            string `json:"provider"`
	RedirectURI         string `json:"redirect_uri"`
	CodeChallenge       string `json:"code_challenge,omitempty"`
	CodeChallengeMethod string `json:"code_challenge_method,omitempty"`
	CodeVerifier        string `json:"code_verifier,omitempty"`
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
	redisClient     *redis.Client
	db              *pgxpool.Pool
	userService     *UserService
	encryptionKey   string
	callbackBaseURL string
}

func NewOAuthService(redisClient *redis.Client, db *pgxpool.Pool, userService *UserService, encryptionKey, callbackBaseURL string) *OAuthService {
	parsed, err := url.Parse(callbackBaseURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		log.Fatalf("Invalid platformURL configured: %v", err)
	}

	return &OAuthService{
		redisClient:     redisClient,
		db:              db,
		userService:     userService,
		encryptionKey:   encryptionKey,
		callbackBaseURL: callbackBaseURL,
	}
}

func (s *OAuthService) getProviderConfig(ctx context.Context, appID, provider string) (*OAuthProviderConfig, error) {
	var clientID, clientSecretEncrypted string
	var scopes []string

	err := s.db.QueryRow(ctx, `
		SELECT client_id, client_secret_encrypted, scopes
		FROM oauth_providers
		WHERE app_id = $1 AND provider = $2 AND enabled = true
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

func (s *OAuthService) InitiateOAuth(ctx context.Context, appID, provider, redirectURI, publicAppID string) (string, string, error) {
	providerConfig, err := s.getProviderConfig(ctx, appID, provider)
	if err != nil {
		return "", "", err
	}

	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	codeVerifier := hex.EncodeToString(verifierBytes)
	challengeHash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(challengeHash[:])
	challengeMethod := "S256"

	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	stateData, err := json.Marshal(oauthState{
		AppID:               appID,
		PublicAppID:         publicAppID,
		Provider:            provider,
		RedirectURI:         redirectURI,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: challengeMethod,
		CodeVerifier:        codeVerifier,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to encode OAuth state: %w", err)
	}
	if err := s.redisClient.Set(ctx, fmt.Sprintf("oauth_state:%s", state), stateData, 10*time.Minute).Err(); err != nil {
		return "", "", fmt.Errorf("failed to store OAuth state: %w", err)
	}

	authURL := s.buildAuthorizationURL(provider, providerConfig.ClientID, state, providerConfig.Scopes, codeChallenge, challengeMethod)
	return authURL, state, nil
}

func (s *OAuthService) HandleOAuthCallback(ctx context.Context, provider, code, state string) (*repository.User, string, string, string, bool, error) {
	// Validate state from Redis
	redisKey := fmt.Sprintf("oauth_state:%s", state)
	stateJSON, err := s.redisClient.Get(ctx, redisKey).Result()
	if err != nil {
		return nil, "", "", "", false, ErrInvalidState
	}

	var stateData oauthState
	if err := json.Unmarshal([]byte(stateJSON), &stateData); err != nil {
		return nil, "", "", "", false, ErrInvalidState
	}

	// Reject path/provider mismatch before consuming state so a retry with the
	// correct callback path can still succeed.
	if provider != stateData.Provider {
		return nil, stateData.AppID, stateData.RedirectURI, stateData.PublicAppID, false, ErrProviderMismatch
	}

	s.redisClient.Del(ctx, redisKey)

	// PKCE Verification
	if stateData.CodeChallenge == "" {
		return nil, stateData.AppID, stateData.RedirectURI, stateData.PublicAppID, false, errors.New("missing PKCE code challenge")
	}
	if !verifyPKCE(stateData.CodeChallenge, stateData.CodeChallengeMethod, stateData.CodeVerifier) {
		return nil, stateData.AppID, stateData.RedirectURI, stateData.PublicAppID, false, ErrInvalidVerifier
	}

	// Get provider config
	providerConfig, err := s.getProviderConfig(ctx, stateData.AppID, provider)
	if err != nil {
		return nil, stateData.AppID, stateData.RedirectURI, stateData.PublicAppID, false, err
	}

	// Exchange code for tokens and get user info
	var email, name, avatarURL, providerUserID string
	var emailVerified bool

	switch provider {
	case "google":
		email, name, avatarURL, providerUserID, emailVerified, err = s.handleGoogleCallback(ctx, code, stateData.CodeVerifier, providerConfig)
	case "github":
		email, name, avatarURL, providerUserID, emailVerified, err = s.handleGithubCallback(ctx, code, stateData.CodeVerifier, providerConfig)
	default:
		return nil, "", "", "", false, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, stateData.AppID, stateData.RedirectURI, stateData.PublicAppID, false, err
	}

	providerUserIDPtr := &providerUserID
	var namePtr, avatarURLPtr *string
	if name != "" {
		namePtr = &name
	}
	if avatarURL != "" {
		avatarURLPtr = &avatarURL
	}

	_, priorErr := s.userService.GetUserByProviderID(ctx, stateData.AppID, provider, providerUserID)
	isNewUser := errors.Is(priorErr, ErrUserNotFound)

	user, err := s.userService.CreateUser(ctx, stateData.AppID, email, namePtr, avatarURLPtr, provider, providerUserIDPtr, nil, emailVerified)
	if err != nil {
		if errors.Is(err, ErrAccountExistsUseOriginalProvider) {
			return nil, stateData.AppID, stateData.RedirectURI, stateData.PublicAppID, false, ErrAccountExistsUseOriginalProvider
		}
		return nil, stateData.AppID, stateData.RedirectURI, stateData.PublicAppID, false, err
	}

	return user, stateData.AppID, stateData.RedirectURI, stateData.PublicAppID, isNewUser, nil
}

func (s *OAuthService) handleGoogleCallback(ctx context.Context, code, codeVerifier string, config *OAuthProviderConfig) (string, string, string, string, bool, error) {
	callbackURL := s.callbackBaseURL + "/oauth/callback/google"
	tokenResp, err := exchangeCodeForToken("https://oauth2.googleapis.com/token", code, codeVerifier, config.ClientID, config.ClientSecret, callbackURL)
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

func (s *OAuthService) handleGithubCallback(ctx context.Context, code, codeVerifier string, config *OAuthProviderConfig) (string, string, string, string, bool, error) {
	callbackURL := s.callbackBaseURL + "/oauth/callback/github"
	tokenResp, err := exchangeCodeForTokenJSON("https://github.com/login/oauth/access_token", code, codeVerifier, config.ClientID, config.ClientSecret, callbackURL)
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

	email, emailVerified, err := s.fetchGithubEmail(ctx, accessToken)
	if err != nil {
		return "", "", "", "", false, err
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
func exchangeCodeForToken(tokenURL, code, codeVerifier, clientID, clientSecret, redirectURI string) (map[string]interface{}, error) {
	resp, err := http.PostForm(tokenURL, map[string][]string{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {codeVerifier},
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
func exchangeCodeForTokenJSON(tokenURL, code, codeVerifier, clientID, clientSecret, redirectURI string) (map[string]interface{}, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	form.Set("redirect_uri", redirectURI)

	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
func (s *OAuthService) buildAuthorizationURL(provider, clientID, state string, scopes []string, codeChallenge, challengeMethod string) string {
	callbackURL := fmt.Sprintf("%s/oauth/callback/%s", s.callbackBaseURL, provider)
	values := url.Values{}
	values.Set("client_id", clientID)
	values.Set("redirect_uri", callbackURL)
	values.Set("state", state)
	values.Set("code_challenge", codeChallenge)
	values.Set("code_challenge_method", challengeMethod)

	switch provider {
	case "google":
		scopeStr := "openid email profile"
		if len(scopes) > 0 {
			scopeStr = strings.Join(scopes, " ")
		}
		values.Set("response_type", "code")
		values.Set("scope", scopeStr)
		values.Set("access_type", "offline")
		return "https://accounts.google.com/o/oauth2/v2/auth?" + values.Encode()
	case "github":
		scopeStr := "user:email"
		if len(scopes) > 0 {
			scopeStr = strings.Join(scopes, " ")
		}
		values.Set("scope", scopeStr)
		return "https://github.com/login/oauth/authorize?" + values.Encode()
	default:
		return ""
	}
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

func verifyPKCE(challenge, method, verifier string) bool {
	if method != "S256" {
		return false
	}
	hash := sha256.Sum256([]byte(verifier))
	actualChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	return actualChallenge == challenge
}
