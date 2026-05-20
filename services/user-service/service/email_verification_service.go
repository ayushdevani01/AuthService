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
func (s *EmailVerificationService) SendVerification(ctx context.Context, appID, userID, email, name, redirectURI string) error {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := hashVerifyToken(rawToken)

	redisKey := fmt.Sprintf("verify:%s:%s", appID, tokenHash)
	s.redis.Set(ctx, redisKey, userID, 24*time.Hour)

	// Keep a short-lived mapping so a second verify after consume can still
	// resolve the user for idempotent success when already verified.
	consumedKey := fmt.Sprintf("verify_consumed:%s:%s", appID, tokenHash)
	s.redis.Set(ctx, consumedKey, userID, 24*time.Hour)

	if err := s.userRepo.StoreEmailVerificationToken(ctx, userID, appID, tokenHash, time.Now().Add(24*time.Hour)); err != nil {
		fmt.Printf("Warning: Failed to store verification token in DB: %v\n", err)
	}

	if name == "" {
		name = email
	}
	return s.emailSvc.SendEmailVerification(ctx, appID, email, name, rawToken, redirectURI)
}

// VerifyEmail validates the token and marks the user's email as verified.
func (s *EmailVerificationService) VerifyEmail(ctx context.Context, appID, rawToken string) (string, error) {
	tokenHash := hashVerifyToken(rawToken)
	redisKey := fmt.Sprintf("verify:%s:%s", appID, tokenHash)
	consumedKey := fmt.Sprintf("verify_consumed:%s:%s", appID, tokenHash)

	userID, err := s.redis.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		userID, err = s.userRepo.GetEmailVerificationToken(ctx, tokenHash, appID)
		if err != nil {
			return "", err
		}
		if userID == "" {
			// Token already consumed: succeed if that user is already verified.
			if consumedUserID, cErr := s.redis.Get(ctx, consumedKey).Result(); cErr == nil && consumedUserID != "" {
				user, findErr := s.userRepo.FindByID(ctx, consumedUserID, appID)
				if findErr == nil && user != nil && user.EmailVerified {
					return consumedUserID, nil
				}
			}
			if anyUserID, aErr := s.userRepo.FindAnyEmailVerificationUser(ctx, tokenHash, appID); aErr == nil && anyUserID != "" {
				user, findErr := s.userRepo.FindByID(ctx, anyUserID, appID)
				if findErr == nil && user != nil && user.EmailVerified {
					return anyUserID, nil
				}
			}
			return "", ErrVerifyTokenInvalid
		}
	} else if err != nil {
		return "", err
	}

	user, findErr := s.userRepo.FindByID(ctx, userID, appID)
	if findErr == nil && user != nil && user.EmailVerified {
		s.redis.Del(ctx, redisKey)
		s.userRepo.DeleteEmailVerificationToken(ctx, tokenHash)
		return userID, nil
	}

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
