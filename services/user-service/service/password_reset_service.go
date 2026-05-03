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
	ErrResetTokenInvalid  = errors.New("reset token is invalid or expired")
	ErrResetTokenUsed     = errors.New("reset token has already been used")
	ErrNoPasswordIdentity = errors.New("no_password_identity")
)

type PasswordResetService struct {
	resetRepo    *repository.PasswordResetRepository
	identityRepo *repository.IdentityRepository
	userRepo     *repository.UserRepository
	sessionRepo  *repository.SessionRepository
	emailSvc     *EmailService
	redis        *redis.Client
}

func NewPasswordResetService(
	resetRepo *repository.PasswordResetRepository,
	identityRepo *repository.IdentityRepository,
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	emailSvc *EmailService,
	redis *redis.Client,
) *PasswordResetService {
	return &PasswordResetService{
		resetRepo:    resetRepo,
		identityRepo: identityRepo,
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		emailSvc:     emailSvc,
		redis:        redis,
	}
}

func (s *PasswordResetService) InitiateReset(ctx context.Context, appID, email, redirectURI string) error {
	user, err := s.userRepo.FindByEmail(ctx, appID, email)
	if err != nil {
		// whether user exists — return success either way
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return err
	}
	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := hashResetToken(rawToken)

	expiresAt := time.Now().Add(15 * time.Minute)

	_, err = s.resetRepo.Create(ctx, user.ID, appID, tokenHash, expiresAt)
	if err != nil {
		return err
	}

	redisKey := fmt.Sprintf("reset:%s", tokenHash)
	s.redis.Set(ctx, redisKey, user.ID, 15*time.Minute)

	name := user.Name
	if name == "" {
		name = email
	}
	return s.emailSvc.SendPasswordReset(ctx, appID, email, name, rawToken, redirectURI)
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, appID, rawToken, newPassword string) error {
	tokenHash := hashResetToken(rawToken)

	record, err := s.resetRepo.FindValidToken(ctx, tokenHash, appID)
	if err != nil {
		return err
	}
	if record == nil {
		anyRecord, statusErr := s.resetRepo.FindByTokenHashAny(ctx, tokenHash)
		if statusErr != nil {
			return statusErr
		}
		if anyRecord == nil || time.Now().After(anyRecord.ExpiresAt) {
			return ErrResetTokenInvalid
		}
		if anyRecord.UsedAt != nil {
			return ErrResetTokenUsed
		}
		return ErrResetTokenInvalid
	}

	_, identErr := s.identityRepo.FindByUserAndProvider(ctx, record.UserID, "email")
	if identErr != nil {
		return ErrNoPasswordIdentity
	}

	hash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}

	rows, err := s.identityRepo.UpdatePasswordHash(ctx, record.UserID, hash)
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrNoPasswordIdentity
	}

	userID, err := s.resetRepo.ConsumeValidToken(ctx, tokenHash, appID)
	if err != nil {
		return err
	}
	if userID == "" {
		return ErrResetTokenInvalid
	}

	redisKey := fmt.Sprintf("reset:%s", tokenHash)
	s.redis.Del(ctx, redisKey)

	if _, err := s.sessionRepo.RevokeAll(ctx, userID, appID, ""); err != nil {
		return err
	}

	return nil
}

func hashResetToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
