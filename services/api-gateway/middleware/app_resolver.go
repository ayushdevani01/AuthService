package middleware

import (
	"context"
	"fmt"
	"time"

	pbDev "github.com/ayushdevan01/AuthService/proto/developer"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type AppResolver struct {
	redisClient *redis.Client
	devClient   pbDev.DeveloperServiceClient
}

func NewAppResolver(redisClient *redis.Client, devClient pbDev.DeveloperServiceClient) *AppResolver {
	return &AppResolver{
		redisClient: redisClient,
		devClient:   devClient,
	}
}

func (r *AppResolver) ResolveAppID(c *gin.Context, publicAppID string) (string, error) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	cacheKey := fmt.Sprintf("app_uuid:%s", publicAppID)

	cached, err := r.redisClient.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		return cached, nil
	}

	resp, err := r.devClient.GetPublicApp(ctx, &pbDev.GetPublicAppRequest{AppId: publicAppID})
	if err != nil {
		return "", err
	}

	if resp.Found && resp.App != nil && resp.App.Id != "" {
		r.redisClient.Set(ctx, cacheKey, resp.App.Id, 24*time.Hour)
		return resp.App.Id, nil
	}

	return "", fmt.Errorf("app not found: %s", publicAppID)
}
