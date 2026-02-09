package service

import (
	"context"
	"errors"

	"github.com/ayushdevan01/AuthService/services/developer-service/auth"
	"github.com/ayushdevan01/AuthService/services/developer-service/repository"
	"github.com/jackc/pgx/v5"
)

var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrDeveloperNotFound  = errors.New("developer not found")
)

type DeveloperService struct {
	repo      *repository.DeveloperRepository
	jwtSecret string
}

func NewDeveloperService(repo *repository.DeveloperRepository, jwtSecret string) *DeveloperService {
	return &DeveloperService{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

type AuthResponse struct {
	Developer   *repository.Developer
	AccessToken string
}

func (s *DeveloperService) Register(ctx context.Context, email, password, name string) (*AuthResponse, error) {
	existing, err := s.repo.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyExists
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	dev, err := s.repo.Create(ctx, email, passwordHash, name)
	if err != nil {
		return nil, err
	}

	token, err := auth.GenerateToken(dev.ID, dev.Email, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Developer:   dev,
		AccessToken: token,
	}, nil
}

func (s *DeveloperService) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
	dev, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if !auth.VerifyPassword(password, dev.PasswordHash) {
		return nil, ErrInvalidCredentials
	}
	token, err := auth.GenerateToken(dev.ID, dev.Email, s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{
		Developer:   dev,
		AccessToken: token,
	}, nil
}

func (s *DeveloperService) GetProfile(ctx context.Context, developerID string) (*repository.Developer, error) {
	dev, err := s.repo.FindByID(ctx, developerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeveloperNotFound
		}
		return nil, err
	}
	return dev, nil
}

func (s *DeveloperService) UpdateProfile(ctx context.Context, developerID string, name *string, password *string) (*repository.Developer, error) {
	var passwordHash *string
	if password != nil {
		hash, err := auth.HashPassword(*password)
		if err != nil {
			return nil, err
		}
		passwordHash = &hash
	}

	dev, err := s.repo.Update(ctx, developerID, name, passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrDeveloperNotFound
		}
		return nil, err
	}
	return dev, nil
}
