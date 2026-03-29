package service

import (
	"context"
	"errors"
	"strconv"

	"github.com/ayushdevan01/AuthService/services/user-service/repository"
	"github.com/jackc/pgx/v5"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
)

type UserService struct {
	userRepo     *repository.UserRepository
	identityRepo *repository.IdentityRepository
}

func NewUserService(userRepo *repository.UserRepository, identityRepo *repository.IdentityRepository) *UserService {
	return &UserService{
		userRepo:     userRepo,
		identityRepo: identityRepo,
	}
}

func (s *UserService) CreateUser(ctx context.Context, appID, email, name, avatarURL, provider string, providerUserID, passwordHash *string, emailVerified bool) (*repository.User, error) {
	// Check if user already exists for this app
	existing, err := s.userRepo.FindByEmail(ctx, appID, email)
	if err == nil && existing != nil {
		// User exists — add identity if new provider
		_, identErr := s.identityRepo.FindByUserAndProvider(ctx, existing.ID, provider)
		if identErr != nil && errors.Is(identErr, pgx.ErrNoRows) {
			// Add new identity to existing user (account linking)
			_, err = s.identityRepo.Create(ctx, existing.ID, provider, providerUserID, passwordHash)
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

func (s *UserService) UpdateUser(ctx context.Context, userID, appID string, email, name, avatarURL *string, emailVerified *bool) (*repository.User, error) {
	user, err := s.userRepo.Update(ctx, userID, appID, email, name, avatarURL, emailVerified)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
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
