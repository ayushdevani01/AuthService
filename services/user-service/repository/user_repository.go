package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID            string
	AppID         string
	Email         string
	Name          string
	AvatarURL     string
	EmailVerified bool
	CreatedAt     time.Time
	LastLoginAt   *time.Time
}

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, appID, email, name, avatarURL string, emailVerified bool) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (app_id, email, name, avatar_url, email_verified)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, app_id, email, name, avatar_url, email_verified, created_at, last_login_at
	`, appID, email, name, avatarURL, emailVerified).Scan(
		&user.ID, &user.AppID, &user.Email, &user.Name, &user.AvatarURL,
		&user.EmailVerified, &user.CreatedAt, &user.LastLoginAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, userID, appID string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, app_id, email, name, avatar_url, email_verified, created_at, last_login_at
		FROM users WHERE id = $1 AND app_id = $2
	`, userID, appID).Scan(
		&user.ID, &user.AppID, &user.Email, &user.Name, &user.AvatarURL,
		&user.EmailVerified, &user.CreatedAt, &user.LastLoginAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, appID, email string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, `
		SELECT id, app_id, email, name, avatar_url, email_verified, created_at, last_login_at
		FROM users WHERE app_id = $1 AND email = $2
	`, appID, email).Scan(
		&user.ID, &user.AppID, &user.Email, &user.Name, &user.AvatarURL,
		&user.EmailVerified, &user.CreatedAt, &user.LastLoginAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET last_login_at = NOW() WHERE id = $1
	`, userID)
	return err
}

func (r *UserRepository) Update(ctx context.Context, userID, appID string, email, name, avatarURL *string, emailVerified *bool) (*User, error) {
	// Fetch current user first
	current, err := r.FindByID(ctx, userID, appID)
	if err != nil {
		return nil, err
	}

	// Apply updates
	newEmail := current.Email
	if email != nil {
		newEmail = *email
	}
	newName := current.Name
	if name != nil {
		newName = *name
	}
	newAvatarURL := current.AvatarURL
	if avatarURL != nil {
		newAvatarURL = *avatarURL
	}
	newEmailVerified := current.EmailVerified
	if emailVerified != nil {
		newEmailVerified = *emailVerified
	}

	user := &User{}
	err = r.db.QueryRow(ctx, `
		UPDATE users SET email = $3, name = $4, avatar_url = $5, email_verified = $6
		WHERE id = $1 AND app_id = $2
		RETURNING id, app_id, email, name, avatar_url, email_verified, created_at, last_login_at
	`, userID, appID, newEmail, newName, newAvatarURL, newEmailVerified).Scan(
		&user.ID, &user.AppID, &user.Email, &user.Name, &user.AvatarURL,
		&user.EmailVerified, &user.CreatedAt, &user.LastLoginAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) Delete(ctx context.Context, userID, appID string) error {
	result, err := r.db.Exec(ctx, `
		DELETE FROM users WHERE id = $1 AND app_id = $2
	`, userID, appID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *UserRepository) List(ctx context.Context, appID string, pageSize int, offset int, providerFilter, emailSearch string) ([]*User, int, error) {
	// Count total
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM users u WHERE u.app_id = $1`
	countArgs := []interface{}{appID}
	argIdx := 2

	if emailSearch != "" {
		countQuery += ` AND u.email ILIKE $` + itoa(argIdx)
		countArgs = append(countArgs, "%"+emailSearch+"%")
		argIdx++
	}

	if providerFilter != "" {
		countQuery += ` AND EXISTS (SELECT 1 FROM user_identities ui WHERE ui.user_id = u.id AND ui.provider = $` + itoa(argIdx) + `)`
		countArgs = append(countArgs, providerFilter)
		argIdx++
	}

	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Fetch page
	query := `SELECT u.id, u.app_id, u.email, u.name, u.avatar_url, u.email_verified, u.created_at, u.last_login_at
		FROM users u WHERE u.app_id = $1`
	args := []interface{}{appID}
	argIdx = 2

	if emailSearch != "" {
		query += ` AND u.email ILIKE $` + itoa(argIdx)
		args = append(args, "%"+emailSearch+"%")
		argIdx++
	}

	if providerFilter != "" {
		query += ` AND EXISTS (SELECT 1 FROM user_identities ui WHERE ui.user_id = u.id AND ui.provider = $` + itoa(argIdx) + `)`
		args = append(args, providerFilter)
		argIdx++
	}

	query += ` ORDER BY u.created_at DESC LIMIT $` + itoa(argIdx) + ` OFFSET $` + itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.AppID, &user.Email, &user.Name, &user.AvatarURL,
			&user.EmailVerified, &user.CreatedAt, &user.LastLoginAt,
		)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, totalCount, nil
}

func itoa(i int) string {
	return string(rune('0' + i)) // works for single digit args (up to 9)
}
