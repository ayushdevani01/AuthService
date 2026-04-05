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
	ErrResetTokenInvalid = errors.New("reset token is invalid or expired")
	ErrResetTokenUsed    = errors.New("reset token has already been used")
)

type PasswordResetService struct {
	resetRepo    *repository.PasswordResetRepository
	identityRepo *repository.IdentityRepository
	userRepo     *repository.UserRepository
	emailSvc     *EmailService
	redis        *redis.Client
}

func NewPasswordResetService(
	resetRepo *repository.PasswordResetRepository,
	identityRepo *repository.IdentityRepository,
	userRepo *repository.UserRepository,
	emailSvc *EmailService,
	redis *redis.Client,
) *PasswordResetService {
	return &PasswordResetService{
		resetRepo:    resetRepo,
		identityRepo: identityRepo,
		userRepo:     userRepo,
		emailSvc:     emailSvc,
		redis:        redis,
	}
}

func (s *PasswordResetService) InitiateReset(ctx context.Context, appID, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, appID, email)
	if err != nil {
		//whether user exists — return success either way
		return nil
	}

	// Generate raw token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := hashResetToken(rawToken)

	expiresAt := time.Now().Add(15 * time.Minute)

	// Store in DB for audit trail
	_, err = s.resetRepo.Create(ctx, user.ID, appID, tokenHash, expiresAt)
	if err != nil {
		return err
	}

	// Cache in Redis for fast lookup (15 min TTL)
	redisKey := fmt.Sprintf("reset:%s", tokenHash)
	s.redis.Set(ctx, redisKey, user.ID, 15*time.Minute)

	// Send the email
	name := user.Name
	if name == "" {
		name = email
	}
	return s.emailSvc.SendPasswordReset(ctx, appID, email, name, rawToken)
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, appID, rawToken, newPassword string) error {
	tokenHash := hashResetToken(rawToken)

	userID, err := s.resetRepo.ConsumeValidToken(ctx, tokenHash)
	if err != nil {
		return err
	}
	if userID == "" {
		record, statusErr := s.resetRepo.FindByTokenHashAny(ctx, tokenHash)
		if statusErr != nil {
			return statusErr
		}
		if record == nil || time.Now().After(record.ExpiresAt) {
			return ErrResetTokenInvalid
		}
		if record.UsedAt != nil {
			return ErrResetTokenUsed
		}
		return ErrResetTokenInvalid
	}

	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	if err := s.identityRepo.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return err
	}

	// Remove from Redis so any cached entry is immediately invalidated
	redisKey := fmt.Sprintf("reset:%s", tokenHash)
	s.redis.Del(ctx, redisKey)

	return nil
}

func hashResetToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
