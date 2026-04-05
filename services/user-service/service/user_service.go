package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/ayushdevan01/AuthService/services/user-service/repository"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserExists       = errors.New("user already exists")
	ErrInvalidPassword  = errors.New("invalid password")
	ErrAccountBlocked   = errors.New("account temporarily blocked due to too many failed login attempts")
	ErrEmailNotVerified = errors.New("email not verified")
)

type UserService struct {
	userRepo     *repository.UserRepository
	identityRepo *repository.IdentityRepository
	rateLimiter  *LoginRateLimiter
}

func NewUserService(userRepo *repository.UserRepository, identityRepo *repository.IdentityRepository, rateLimiter *LoginRateLimiter) *UserService {
	return &UserService{
		userRepo:     userRepo,
		identityRepo: identityRepo,
		rateLimiter:  rateLimiter,
	}
}

func (s *UserService) CreateUser(ctx context.Context, appID, email string, name, avatarURL *string, provider string, providerUserID, passwordHash *string, emailVerified bool) (*repository.User, error) {
	// Check if user already exists for this app
	existing, err := s.userRepo.FindByEmail(ctx, appID, email)
	if err == nil && existing != nil {
		// User exists — add identity if new provider
		_, identErr := s.identityRepo.FindByUserAndProvider(ctx, existing.ID, provider)
		if identErr != nil {
			if errors.Is(identErr, pgx.ErrNoRows) {
				// Add new identity to existing user (account linking)
				_, err = s.identityRepo.Create(ctx, existing.ID, provider, providerUserID, passwordHash)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, identErr
			}
		}

		var updateName, updateAvatarURL *string
		if name != nil && *name != "" && existing.Name == "" {
			updateName = name
		}
		if avatarURL != nil && *avatarURL != "" && existing.AvatarURL == "" {
			updateAvatarURL = avatarURL
		}

		// If OAuth provider confirms email is verified but user is not, update it
		var updateEmailVerified *bool
		if emailVerified && !existing.EmailVerified {
			updateEmailVerified = &emailVerified
		}

		if updateName != nil || updateAvatarURL != nil || updateEmailVerified != nil {
			existing, err = s.userRepo.Update(ctx, existing.ID, existing.AppID, nil, updateName, updateAvatarURL, updateEmailVerified)
			if err != nil {
				return nil, err
			}
		}

		// Update last login
		s.userRepo.UpdateLastLogin(ctx, existing.ID)
		return existing, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	// Create new user
	user, err := s.userRepo.Create(ctx, appID, email, name, avatarURL, emailVerified)
	if err != nil {
		return nil, err
	}

	// Create identity
	_, err = s.identityRepo.Create(ctx, user.ID, provider, providerUserID, passwordHash)
	if err != nil {
		return nil, err
	}
	user.Provider = provider
	if providerUserID != nil {
		user.ProviderUserID = *providerUserID
	}

	return user, nil
}

func (s *UserService) GetUser(ctx context.Context, userID, appID string) (*repository.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetUserByEmail(ctx context.Context, appID, email string) (*repository.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, appID, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (s *UserService) GetUserByProviderID(ctx context.Context, appID, provider, providerUserID string) (*repository.User, error) {
	identity, err := s.identityRepo.FindByProviderID(ctx, provider, providerUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, identity.UserID, appID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, userID, appID string, email, passwordHash, name, avatarURL *string, emailVerified *bool) (*repository.User, error) {
	user, err := s.userRepo.Update(ctx, userID, appID, email, name, avatarURL, emailVerified)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if passwordHash != nil {
		if err := s.identityRepo.UpdatePasswordHash(ctx, userID, *passwordHash); err != nil {
			return nil, err
		}
	}
	return user, nil
}

func (s *UserService) DeleteUser(ctx context.Context, userID, appID string) error {
	err := s.userRepo.Delete(ctx, userID, appID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *UserService) ListUsers(ctx context.Context, appID string, pageSize int, pageToken, providerFilter, emailSearch string) ([]*repository.User, string, int, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := 0
	if pageToken != "" {
		var err error
		offset, err = strconv.Atoi(pageToken)
		if err != nil {
			offset = 0
		}
	}

	users, totalCount, err := s.userRepo.List(ctx, appID, pageSize, offset, providerFilter, emailSearch)
	if err != nil {
		return nil, "", 0, err
	}

	nextPageToken := ""
	if offset+pageSize < totalCount {
		nextPageToken = strconv.Itoa(offset + pageSize)
	}

	return users, nextPageToken, totalCount, nil
}

func (s *UserService) RegisterWithEmail(ctx context.Context, appID, email, password, name string) (*repository.User, error) {
	// Check if user already has an email identity for this app
	existing, err := s.userRepo.FindByEmail(ctx, appID, email)
	if err == nil && existing != nil {
		_, identErr := s.identityRepo.FindByUserAndProvider(ctx, existing.ID, "email")
		if identErr == nil {
			return nil, ErrUserExists
		}
		if !errors.Is(identErr, pgx.ErrNoRows) {
			return nil, identErr
		}

		// If they don't have an email identity, link the new email/password to this account
		hash, hashErr := hashPassword(password)
		if hashErr != nil {
			return nil, hashErr
		}
		_, createErr := s.identityRepo.Create(ctx, existing.ID, "email", nil, &hash)
		if createErr != nil {
			return nil, createErr
		}
		return existing, nil
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	emptyAvatar := ""
	user, err := s.userRepo.Create(ctx, appID, email, &name, &emptyAvatar, false)
	if err != nil {
		return nil, err
	}

	_, err = s.identityRepo.Create(ctx, user.ID, "email", nil, &hash)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) LoginWithEmail(ctx context.Context, appID, email, password string, requireEmailVerification bool) (*repository.User, error) {
	blocked, err := s.rateLimiter.IsBlocked(ctx, appID, email)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, ErrAccountBlocked
	}

	user, err := s.userRepo.FindByEmail(ctx, appID, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			s.rateLimiter.RecordFailure(ctx, appID, email)
			return nil, ErrInvalidPassword
		}
		return nil, err
	}

	identity, err := s.identityRepo.FindByUserAndProvider(ctx, user.ID, "email")
	if err != nil || identity.PasswordHash == nil {
		s.rateLimiter.RecordFailure(ctx, appID, email)
		return nil, ErrInvalidPassword
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*identity.PasswordHash), []byte(password)); err != nil {
		s.rateLimiter.RecordFailure(ctx, appID, email)
		return nil, ErrInvalidPassword
	}

	if requireEmailVerification && !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	s.rateLimiter.ClearFailures(ctx, appID, email)
	s.userRepo.UpdateLastLogin(ctx, user.ID)
	return user, nil
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
