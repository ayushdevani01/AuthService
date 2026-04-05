package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/ayushdevan01/AuthService/services/token-service/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrNoSigningKey        = errors.New("no active signing key for app")
	ErrInvalidPrivateKey   = errors.New("invalid private key")
)

type TokenService struct {
	signingKeyRepo *repository.SigningKeyRepository
	sessionRepo    *repository.SessionRepository
}

func NewTokenService(signingKeyRepo *repository.SigningKeyRepository, sessionRepo *repository.SessionRepository) *TokenService {
	return &TokenService{
		signingKeyRepo: signingKeyRepo,
		sessionRepo:    sessionRepo,
	}
}

type TokenPairResult struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  int64
	RefreshTokenExpiresAt int64
}

func (s *TokenService) GenerateTokenPair(ctx context.Context, appID, userID, email, provider string, emailVerified bool, sessionID string, accessTTL, refreshTTL int64) (*TokenPairResult, error) {
	// Refresh token is created by user-service CreateSession.
	// This method is only responsible for generating the access token and optionally rotating the refresh token.

	// Set defaults
	if accessTTL <= 0 {
		accessTTL = 3600 // 1 hour
	}

	// Get signing key
	privateKeyPEM, kid, _, err := s.signingKeyRepo.GetDecryptedPrivateKey(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoSigningKey, err)
	}

	// Parse private key
	rsaKey, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	accessExp := now.Add(time.Duration(accessTTL) * time.Second)
	refreshExp := now.Add(time.Duration(refreshTTL) * time.Second)

	// Build JWT claims
	claims := jwt.MapClaims{
		"iss":            "https://auth.yourplatform.com",
		"sub":            userID,
		"aud":            appID,
		"exp":            accessExp.Unix(),
		"iat":            now.Unix(),
		"email":          email,
		"provider":       provider,
		"email_verified": emailVerified,
	}

	// Sign with RS256
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	accessToken, err := token.SignedString(rsaKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	return &TokenPairResult{
		AccessToken:           accessToken,
		RefreshToken:          "",
		AccessTokenExpiresAt:  accessExp.Unix(),
		RefreshTokenExpiresAt: refreshExp.Unix(),
	}, nil
}

func (s *TokenService) RefreshTokens(ctx context.Context, refreshToken, appID string, rotateRefreshToken bool) (string, *string, int64, *int64, error) {
	// Get signing key to sign the new access token
	privateKeyPEM, kid, _, err := s.signingKeyRepo.GetDecryptedPrivateKey(ctx, appID)
	if err != nil {
		return "", nil, 0, nil, fmt.Errorf("%w: %v", ErrNoSigningKey, err)
	}

	// Hash the refresh token and find the session
	refreshHash := repository.HashToken(refreshToken)
	session, err := s.sessionRepo.FindByRefreshTokenHash(ctx, refreshHash)
	if err != nil {
		return "", nil, 0, nil, ErrInvalidRefreshToken
	}
	if session.AppID != appID {
		return "", nil, 0, nil, ErrInvalidRefreshToken
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return "", nil, 0, nil, ErrInvalidRefreshToken
	}

	// Get user info for new access token (using internal appID)
	email, provider, emailVerified, err := s.sessionRepo.GetUserInfo(ctx, session.UserID, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil, 0, nil, ErrInvalidRefreshToken
		}
		return "", nil, 0, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	rsaKey, err := parseRSAPrivateKey(privateKeyPEM)
	if err != nil {
		return "", nil, 0, nil, err
	}

	// Sign new access token
	now := time.Now()
	accessExp := now.Add(1 * time.Hour)

	claims := jwt.MapClaims{
		"iss":            "https://auth.yourplatform.com",
		"sub":            session.UserID,
		"aud":            appID,
		"exp":            accessExp.Unix(),
		"iat":            now.Unix(),
		"email":          email,
		"provider":       provider,
		"email_verified": emailVerified,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid

	newAccessToken, err := token.SignedString(rsaKey)
	if err != nil {
		return "", nil, 0, nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Optionally rotate refresh token
	var newRefreshToken *string
	var newRefreshExp *int64
	if rotateRefreshToken {
		refreshBytes := make([]byte, 32)
		if _, err := rand.Read(refreshBytes); err != nil {
			return "", nil, 0, nil, err
		}
		rt := hex.EncodeToString(refreshBytes)
		newRefreshToken = &rt

		newHash := repository.HashToken(rt)
		newExpTime := now.Add(30 * 24 * time.Hour)
		exp := newExpTime.Unix()
		newRefreshExp = &exp

		if err := s.sessionRepo.UpdateRefreshToken(ctx, session.ID, newHash, newExpTime); err != nil {
			return "", nil, 0, nil, fmt.Errorf("failed to rotate refresh token: %w", err)
		}
	}

	return newAccessToken, newRefreshToken, accessExp.Unix(), newRefreshExp, nil
}

func (s *TokenService) RevokeToken(ctx context.Context, token, tokenType, appID string) error {
	if tokenType == "refresh" {
		hash := repository.HashToken(token)
		session, err := s.sessionRepo.FindByRefreshTokenHash(ctx, hash)
		if err != nil {
			return ErrInvalidRefreshToken
		}
		if session.AppID != appID {
			return ErrInvalidRefreshToken
		}
		return s.sessionRepo.Revoke(ctx, session.ID)
	}
	return errors.New("access tokens cannot be revoked directly")
}

type PublicKeyInfo struct {
	KID       string
	PublicKey string
	IsActive  bool
}

func (s *TokenService) GetPublicKeys(ctx context.Context, appID string) ([]*PublicKeyInfo, error) {
	keys, err := s.signingKeyRepo.ListPublicKeys(ctx, appID)
	if err != nil {
		return nil, err
	}

	result := make([]*PublicKeyInfo, 0, len(keys))
	for _, k := range keys {
		result = append(result, &PublicKeyInfo{
			KID:       k.KID,
			PublicKey: k.PublicKey,
			IsActive:  k.IsActive,
		})
	}
	return result, nil
}

func parseRSAPrivateKey(pemStr string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, ErrInvalidPrivateKey
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8
		keyInterface, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
		}
		rsaKey, ok := keyInterface.(*rsa.PrivateKey)
		if !ok {
			return nil, ErrInvalidPrivateKey
		}
		return rsaKey, nil
	}

	return key, nil
}
