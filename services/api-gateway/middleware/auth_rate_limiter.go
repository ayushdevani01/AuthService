package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// AuthRateLimiter applies fixed-window request limits keyed by app + client IP.
type AuthRateLimiter struct {
	redis *redis.Client
}

func NewAuthRateLimiter(r *redis.Client) *AuthRateLimiter {
	return &AuthRateLimiter{redis: r}
}

func (l *AuthRateLimiter) allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	script := `
		local c = redis.call('INCR', KEYS[1])
		if c == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[1])
		end
		return c
	`
	count, err := l.redis.Eval(ctx, script, []string{key}, int(window.Seconds())).Int()
	if err != nil {
		return false, err
	}
	return count <= limit, nil
}

func (l *AuthRateLimiter) key(kind, appID, ip string) string {
	return fmt.Sprintf("auth_rl:%s:%s:%s", kind, appID, ip)
}

// AllowRegister caps register attempts per app+IP (20 / 15 min).
func (l *AuthRateLimiter) AllowRegister(ctx context.Context, appID, ip string) (bool, error) {
	return l.allow(ctx, l.key("register", appID, ip), 20, 15*time.Minute)
}

// AllowForgotPassword caps forgot-password attempts per app+IP (5 / 15 min).
func (l *AuthRateLimiter) AllowForgotPassword(ctx context.Context, appID, ip string) (bool, error) {
	return l.allow(ctx, l.key("forgot", appID, ip), 5, 15*time.Minute)
}
