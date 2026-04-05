package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/ayushdevan01/AuthService/services/user-service/repository"
)

type SessionService struct {
	sessionRepo *repository.SessionRepository
}

func NewSessionService(sessionRepo *repository.SessionRepository) *SessionService {
	return &SessionService{sessionRepo: sessionRepo}
}

type SessionResult struct {
	Session      *repository.Session
	RefreshToken string
}

func (s *SessionService) CreateSession(ctx context.Context, userID, appID, userAgent, ipAddress string, ttlSeconds int64) (*SessionResult, error) {
	// Generate refresh token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	refreshToken := hex.EncodeToString(tokenBytes)
	refreshTokenHash := repository.HashToken(refreshToken)

	if ttlSeconds <= 0 {
		ttlSeconds = 30 * 24 * 60 * 60 // 30 days default
	}
	expiresAt := time.Now().Add(time.Duration(ttlSeconds) * time.Second)

	session, err := s.sessionRepo.Create(ctx, userID, appID, refreshTokenHash, userAgent, ipAddress, expiresAt)
	if err != nil {
		return nil, err
	}

	return &SessionResult{
		Session:      session,
		RefreshToken: refreshToken,
	}, nil
}

func (s *SessionService) GetSession(ctx context.Context, refreshTokenHash string) (*repository.Session, bool, bool, error) {
	session, err := s.sessionRepo.FindByRefreshTokenHash(ctx, refreshTokenHash)
	if err != nil {
		return nil, false, false, err
	}

	expired := time.Now().After(session.ExpiresAt)
	return session, true, expired, nil
}

func (s *SessionService) RevokeSession(ctx context.Context, userID, sessionID string) error {
	return s.sessionRepo.Revoke(ctx, userID, sessionID)
}

func (s *SessionService) RevokeAllSessions(ctx context.Context, userID, appID, exceptSessionID string) (int, error) {
	return s.sessionRepo.RevokeAll(ctx, userID, appID, exceptSessionID)
}

func (s *SessionService) ListSessions(ctx context.Context, userID, appID string) ([]*repository.Session, error) {
	return s.sessionRepo.ListByUser(ctx, userID, appID)
}
