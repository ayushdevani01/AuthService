package service

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// LoginRateLimiter blocks login after 10 consecutive failures within 15 minutes.
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
	script := `
		local c = redis.call('INCR', KEYS[1])
		redis.call('EXPIRE', KEYS[1], ARGV[1])
		return c
	`
	_, err := l.redis.Eval(ctx, script, []string{k}, 15*60).Result()
	return err
}

// ClearFailures resets the counter after a successful login.
func (l *LoginRateLimiter) ClearFailures(ctx context.Context, appID, email string) {
	l.redis.Del(ctx, l.key(appID, email))
}
