package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ayushdevan01/AuthService/services/user-service/repository"
	"github.com/redis/go-redis/v9"
)

var (
	ErrVerifyTokenInvalid = errors.New("verification token is invalid or expired")
)

type EmailVerificationService struct {
	userRepo *repository.UserRepository
	emailSvc *EmailService
	redis    *redis.Client
}

func NewEmailVerificationService(
	userRepo *repository.UserRepository,
	emailSvc *EmailService,
	redis *redis.Client,
) *EmailVerificationService {
	return &EmailVerificationService{
		userRepo: userRepo,
		emailSvc: emailSvc,
		redis:    redis,
	}
}

// SendVerification generates a secure token and sends a verification email.
func (s *EmailVerificationService) SendVerification(ctx context.Context, appID, userID, email, name string) error {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := hashVerifyToken(rawToken)

	redisKey := fmt.Sprintf("verify:%s:%s", appID, tokenHash)
	s.redis.Set(ctx, redisKey, userID, 24*time.Hour)

	// DB Fallback
	if err := s.userRepo.StoreEmailVerificationToken(ctx, userID, appID, tokenHash, time.Now().Add(24*time.Hour)); err != nil {
		// Log error but don't fail, redis might still succeed
		fmt.Printf("Warning: Failed to store verification token in DB: %v\n", err)
	}

	if name == "" {
		name = email
	}
	return s.emailSvc.SendEmailVerification(ctx, appID, email, name, rawToken)
}

// VerifyEmail validates the token and marks the user's email as verified.
func (s *EmailVerificationService) VerifyEmail(ctx context.Context, appID, rawToken string) (string, error) {
	tokenHash := hashVerifyToken(rawToken)
	redisKey := fmt.Sprintf("verify:%s:%s", appID, tokenHash)

	userID, err := s.redis.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		// Fallback to DB
		userID, err = s.userRepo.GetEmailVerificationToken(ctx, tokenHash, appID)
		if err != nil {
			return "", err
		}
		if userID == "" {
			return "", ErrVerifyTokenInvalid
		}
	} else if err != nil {
		return "", err
	}

	// Mark email_verified = true
	emailVerified := true
	_, err = s.userRepo.Update(ctx, userID, appID, nil, nil, nil, &emailVerified)
	if err != nil {
		return "", err
	}

	s.redis.Del(ctx, redisKey)
	s.userRepo.DeleteEmailVerificationToken(ctx, tokenHash)

	return userID, nil
}

func hashVerifyToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
