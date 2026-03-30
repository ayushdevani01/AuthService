package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// LoginRateLimiter blocks login after 5 consecutive failures within 15 minutes.
type LoginRateLimiter struct {
	redis *redis.Client
}

func NewLoginRateLimiter(r *redis.Client) *LoginRateLimiter {
	return &LoginRateLimiter{redis: r}
}

func (l *LoginRateLimiter) key(appID, email string) string {
	return fmt.Sprintf("login_attempts:%s:%s", appID, email)
}

// IsBlocked returns true if this app+email combo has too many recent failures.
func (l *LoginRateLimiter) IsBlocked(ctx context.Context, appID, email string) (bool, error) {
	val, err := l.redis.Get(ctx, l.key(appID, email)).Int()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val >= 10, nil
}

// RecordFailure increments the failure counter. Resets TTL to 15 min on each failure.
func (l *LoginRateLimiter) RecordFailure(ctx context.Context, appID, email string) error {
	k := l.key(appID, email)
	pipe := l.redis.Pipeline()
	pipe.Incr(ctx, k)
	pipe.Expire(ctx, k, 15*time.Minute)
	_, err := pipe.Exec(ctx)
	return err
}

// ClearFailures resets the counter after a successful login.
func (l *LoginRateLimiter) ClearFailures(ctx context.Context, appID, email string) {
	l.redis.Del(ctx, l.key(appID, email))
}
