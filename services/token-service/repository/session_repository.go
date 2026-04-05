package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Session struct {
	ID               string
	UserID           string
	AppID            string
	RefreshTokenHash string
	UserAgent        string
	IPAddress        string
	CreatedAt        time.Time
	ExpiresAt        time.Time
	RevokedAt        *time.Time
}

type sessionCache struct {
	UserID    string `json:"user_id"`
	AppID     string `json:"app_id"`
	ExpiresAt int64  `json:"expires_at"`
}

type SessionRepository struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewSessionRepository(db *pgxpool.Pool, redis *redis.Client) *SessionRepository {
	return &SessionRepository{db: db, redis: redis}
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (r *SessionRepository) FindByRefreshTokenHash(ctx context.Context, hash string) (*Session, error) {
	redisKey := fmt.Sprintf("session:%s", hash)

	// Try Redis first (fast path)
	cached, err := r.redis.Get(ctx, redisKey).Result()
	if err == nil {
		var sc sessionCache
		if json.Unmarshal([]byte(cached), &sc) == nil {
			// Found in cache — fetch full session from Postgres
			session := &Session{}
			err := r.db.QueryRow(ctx, `
				SELECT id, user_id, app_id, refresh_token_hash, user_agent, ip_address, created_at, expires_at, revoked_at
				FROM sessions WHERE refresh_token_hash = $1 AND revoked_at IS NULL
			`, hash).Scan(
				&session.ID, &session.UserID, &session.AppID, &session.RefreshTokenHash,
				&session.UserAgent, &session.IPAddress, &session.CreatedAt, &session.ExpiresAt, &session.RevokedAt,
			)
			if err != nil {
				return nil, err
			}
			return session, nil
		}
	}

	// Fallback to Postgres
	session := &Session{}
	err = r.db.QueryRow(ctx, `
		SELECT id, user_id, app_id, refresh_token_hash, user_agent, ip_address, created_at, expires_at, revoked_at
		FROM sessions WHERE refresh_token_hash = $1 AND revoked_at IS NULL
	`, hash).Scan(
		&session.ID, &session.UserID, &session.AppID, &session.RefreshTokenHash,
		&session.UserAgent, &session.IPAddress, &session.CreatedAt, &session.ExpiresAt, &session.RevokedAt,
	)
	if err != nil {
		return nil, err
	}

	// Re-cache in Redis
	cacheData, _ := json.Marshal(sessionCache{
		UserID:    session.UserID,
		AppID:     session.AppID,
		ExpiresAt: session.ExpiresAt.Unix(),
	})
	ttl := time.Until(session.ExpiresAt)
	if ttl > 0 {
		r.redis.Set(ctx, redisKey, cacheData, ttl)
	}

	return session, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, sessionID string) error {
	var refreshTokenHash string
	err := r.db.QueryRow(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING refresh_token_hash
	`, sessionID).Scan(&refreshTokenHash)
	if err != nil {
		return err
	}

	r.redis.Del(ctx, "session:"+refreshTokenHash)
	return nil
}

func (r *SessionRepository) UpdateRefreshToken(ctx context.Context, sessionID, newHash string, newExpiresAt time.Time) error {
	// Get old hash first so we can clean up Redis
	var oldHash string
	var userID, appID string
	err := r.db.QueryRow(ctx, `
		SELECT refresh_token_hash, user_id, app_id FROM sessions
		WHERE id = $1 AND revoked_at IS NULL
	`, sessionID).Scan(&oldHash, &userID, &appID)
	if err != nil {
		return err
	}

	// Update in Postgres
	_, err = r.db.Exec(ctx, `
		UPDATE sessions SET refresh_token_hash = $2, expires_at = $3
		WHERE id = $1 AND revoked_at IS NULL
	`, sessionID, newHash, newExpiresAt)
	if err != nil {
		return err
	}

	// Remove old Redis cache entry
	r.redis.Del(ctx, fmt.Sprintf("session:%s", oldHash))

	// Set new Redis cache entry
	cacheData, _ := json.Marshal(sessionCache{
		UserID:    userID,
		AppID:     appID,
		ExpiresAt: newExpiresAt.Unix(),
	})
	ttl := time.Until(newExpiresAt)
	if ttl > 0 {
		r.redis.Set(ctx, fmt.Sprintf("session:%s", newHash), cacheData, ttl)
	}

	return nil
}

func (r *SessionRepository) GetUserInfo(ctx context.Context, userID, appID string) (string, string, bool, error) {
	var email, provider string
	var emailVerified bool
	err := r.db.QueryRow(ctx, `
		SELECT u.email, COALESCE(ui.provider, 'unknown'), u.email_verified
		FROM users u
		LEFT JOIN user_identities ui ON ui.user_id = u.id
		WHERE u.id = $1 AND u.app_id = $2
		ORDER BY ui.created_at LIMIT 1
	`, userID, appID).Scan(&email, &provider, &emailVerified)
	if err != nil {
		return "", "", false, err
	}
	return email, provider, emailVerified, nil
}
