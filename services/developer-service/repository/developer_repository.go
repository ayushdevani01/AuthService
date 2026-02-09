package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Developer struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type DeveloperRepository struct {
	db *pgxpool.Pool
}

func NewDeveloperRepository(db *pgxpool.Pool) *DeveloperRepository {
	return &DeveloperRepository{db: db}
}

func (r *DeveloperRepository) Create(ctx context.Context, email, passwordHash, name string) (*Developer, error) {
	dev := &Developer{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO developers (email, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, name, created_at, updated_at
	`, email, passwordHash, name).Scan(
		&dev.ID, &dev.Email, &dev.PasswordHash, &dev.Name, &dev.CreatedAt, &dev.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

func (r *DeveloperRepository) FindByEmail(ctx context.Context, email string) (*Developer, error) {
	dev := &Developer{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM developers
		WHERE email = $1
	`, email).Scan(
		&dev.ID, &dev.Email, &dev.PasswordHash, &dev.Name, &dev.CreatedAt, &dev.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

func (r *DeveloperRepository) FindByID(ctx context.Context, id string) (*Developer, error) {
	dev := &Developer{}
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM developers
		WHERE id = $1
	`, id).Scan(
		&dev.ID, &dev.Email, &dev.PasswordHash, &dev.Name, &dev.CreatedAt, &dev.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return dev, nil
}

func (r *DeveloperRepository) Update(ctx context.Context, id string, name *string, passwordHash *string) (*Developer, error) {
	dev := &Developer{}

	// dynamic update query
	if name != nil && passwordHash != nil {
		err := r.db.QueryRow(ctx, `
			UPDATE developers SET name = $2, password_hash = $3, updated_at = NOW()
			WHERE id = $1
			RETURNING id, email, password_hash, name, created_at, updated_at
		`, id, *name, *passwordHash).Scan(
			&dev.ID, &dev.Email, &dev.PasswordHash, &dev.Name, &dev.CreatedAt, &dev.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
	} else if name != nil {
		err := r.db.QueryRow(ctx, `
			UPDATE developers SET name = $2, updated_at = NOW()
			WHERE id = $1
			RETURNING id, email, password_hash, name, created_at, updated_at
		`, id, *name).Scan(
			&dev.ID, &dev.Email, &dev.PasswordHash, &dev.Name, &dev.CreatedAt, &dev.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
	} else if passwordHash != nil {
		err := r.db.QueryRow(ctx, `
			UPDATE developers SET password_hash = $2, updated_at = NOW()
			WHERE id = $1
			RETURNING id, email, password_hash, name, created_at, updated_at
		`, id, *passwordHash).Scan(
			&dev.ID, &dev.Email, &dev.PasswordHash, &dev.Name, &dev.CreatedAt, &dev.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
	} else {
		return r.FindByID(ctx, id)
	}

	return dev, nil
}
