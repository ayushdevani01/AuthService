package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

type sessionCache struct {
	UserID    string `json:"user_id"`
	AppID     string `json:"app_id"`
	ExpiresAt int64  `json:"expires_at"`
}

func (r *SessionRepository) Create(ctx context.Context, userID, appID, refreshTokenHash, userAgent, ipAddress string, expiresAt time.Time) (*Session, error) {
	session := &Session{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO sessions (user_id, app_id, refresh_token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, app_id, refresh_token_hash, user_agent, ip_address, created_at, expires_at, revoked_at
	`, userID, appID, refreshTokenHash, userAgent, ipAddress, expiresAt).Scan(
		&session.ID, &session.UserID, &session.AppID, &session.RefreshTokenHash,
		&session.UserAgent, &session.IPAddress, &session.CreatedAt, &session.ExpiresAt, &session.RevokedAt,
	)
	if err != nil {
		return nil, err
	}

	// Cache in Redis
	cacheData, _ := json.Marshal(sessionCache{
		UserID:    userID,
		AppID:     appID,
		ExpiresAt: expiresAt.Unix(),
	})
	redisKey := fmt.Sprintf("session:%s", refreshTokenHash)
	ttl := time.Until(expiresAt)
	r.redis.Set(ctx, redisKey, cacheData, ttl)

	return session, nil
}

func (r *SessionRepository) FindByRefreshTokenHash(ctx context.Context, hash string) (*Session, error) {
	// Try Redis first
	redisKey := fmt.Sprintf("session:%s", hash)
	cached, err := r.redis.Get(ctx, redisKey).Result()
	if err == nil {
		var sc sessionCache
		if json.Unmarshal([]byte(cached), &sc) == nil {
			// Found in Redis, get full session from Postgres
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

func (r *SessionRepository) Revoke(ctx context.Context, sessionID, userID string) error {
	// Get the session to find the refresh token hash for Redis cleanup
	var refreshTokenHash string
	err := r.db.QueryRow(ctx, `
		UPDATE sessions SET revoked_at = NOW()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
		RETURNING refresh_token_hash
	`, sessionID, userID).Scan(&refreshTokenHash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("session not found")
		}
		return err
	}

	// Remove from Redis
	redisKey := fmt.Sprintf("session:%s", refreshTokenHash)
	r.redis.Del(ctx, redisKey)

	return nil
}

func (r *SessionRepository) RevokeAll(ctx context.Context, userID, appID, exceptSessionID string) (int, error) {
	var result int

	query := `
		UPDATE sessions SET revoked_at = NOW()
		WHERE user_id = $1 AND app_id = $2 AND revoked_at IS NULL
	`
	args := []interface{}{userID, appID}

	if exceptSessionID != "" {
		query += ` AND id != $3`
		args = append(args, exceptSessionID)
	}

	// First get all tokens to remove from Redis
	selectQuery := `
		SELECT refresh_token_hash FROM sessions
		WHERE user_id = $1 AND app_id = $2 AND revoked_at IS NULL
	`
	selectArgs := []interface{}{userID, appID}
	if exceptSessionID != "" {
		selectQuery += ` AND id != $3`
		selectArgs = append(selectArgs, exceptSessionID)
	}

	rows, err := r.db.Query(ctx, selectQuery, selectArgs...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var hash string
		if err := rows.Scan(&hash); err != nil {
			return 0, err
		}
		hashes = append(hashes, hash)
	}

	// Revoke in Postgres
	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	result = int(tag.RowsAffected())

	// Remove all from Redis
	for _, hash := range hashes {
		r.redis.Del(ctx, fmt.Sprintf("session:%s", hash))
	}

	return result, nil
}

func (r *SessionRepository) ListByUser(ctx context.Context, userID, appID string) ([]*Session, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, app_id, refresh_token_hash, user_agent, ip_address, created_at, expires_at, revoked_at
		FROM sessions WHERE user_id = $1 AND app_id = $2 AND revoked_at IS NULL AND expires_at > NOW()
		ORDER BY created_at DESC
	`, userID, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		s := &Session{}
		err := rows.Scan(
			&s.ID, &s.UserID, &s.AppID, &s.RefreshTokenHash,
			&s.UserAgent, &s.IPAddress, &s.CreatedAt, &s.ExpiresAt, &s.RevokedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (r *SessionRepository) UpdateRefreshToken(ctx context.Context, sessionID, newHash string, newExpiresAt time.Time) (string, error) {
	var oldHash string
	err := r.db.QueryRow(ctx, `
		UPDATE sessions SET refresh_token_hash = $2, expires_at = $3
		WHERE id = $1 AND revoked_at IS NULL
		RETURNING (SELECT refresh_token_hash FROM sessions WHERE id = $1)
	`, sessionID, newHash, newExpiresAt).Scan(&oldHash)
	if err != nil {
		return "", err
	}

	// Remove old Redis key, set new one
	r.redis.Del(ctx, fmt.Sprintf("session:%s", oldHash))

	var userID, appID string
	r.db.QueryRow(ctx, `SELECT user_id, app_id FROM sessions WHERE id = $1`, sessionID).Scan(&userID, &appID)

	cacheData, _ := json.Marshal(sessionCache{
		UserID:    userID,
		AppID:     appID,
		ExpiresAt: newExpiresAt.Unix(),
	})
	ttl := time.Until(newExpiresAt)
	if ttl > 0 {
		r.redis.Set(ctx, fmt.Sprintf("session:%s", newHash), cacheData, ttl)
	}

	return oldHash, nil
}
