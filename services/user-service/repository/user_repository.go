package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID             string
	AppID          string
	Email          string
	Name           string
	AvatarURL      string
	Provider       string
	ProviderUserID string
	EmailVerified  bool
	CreatedAt      time.Time
	LastLoginAt    *time.Time
}

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, appID, email string, name, avatarURL *string, emailVerified bool) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (app_id, email, name, avatar_url, email_verified)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, app_id, email, COALESCE(name, ''), COALESCE(avatar_url, ''), email_verified, created_at, last_login_at
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
		SELECT id, app_id, email, COALESCE(name, ''), COALESCE(avatar_url, ''),
		       email_verified, created_at, last_login_at
		FROM users
		WHERE id = $1 AND app_id = $2
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
		SELECT id, app_id, email, COALESCE(name, ''), COALESCE(avatar_url, ''),
		       email_verified, created_at, last_login_at
		FROM users
		WHERE app_id = $1 AND email = $2
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
	user := &User{}
	err := r.db.QueryRow(ctx, `
		UPDATE users
		SET
			email = COALESCE($3, email),
			name = COALESCE($4, name),
			avatar_url = COALESCE($5, avatar_url),
			email_verified = COALESCE($6, email_verified)
		WHERE id = $1 AND app_id = $2
		RETURNING id, app_id, email, COALESCE(name, ''), COALESCE(avatar_url, ''),
		          email_verified, created_at, last_login_at
	`, userID, appID, email, name, avatarURL, emailVerified).Scan(
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
	var totalCount int
	countQuery := `SELECT COUNT(*) FROM users WHERE app_id = $1`
	countArgs := []interface{}{appID}
	argIdx := 2

	if emailSearch != "" {
		escapedSearch := strings.ReplaceAll(emailSearch, `\`, `\\`)
		escapedSearch = strings.ReplaceAll(escapedSearch, `%`, `\%`)
		escapedSearch = strings.ReplaceAll(escapedSearch, `_`, `\_`)
		countQuery += ` AND email ILIKE $` + itoa(argIdx)
		countArgs = append(countArgs, "%"+escapedSearch+"%")
		argIdx++
	}

	if providerFilter != "" {
		countQuery += ` AND EXISTS (SELECT 1 FROM user_identities WHERE user_id = users.id AND provider = $` + itoa(argIdx) + `)`
		countArgs = append(countArgs, providerFilter)
		argIdx++
	}

	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	query := `SELECT id, app_id, email, COALESCE(name, ''), COALESCE(avatar_url, ''),
	                 email_verified, created_at, last_login_at
	          FROM users WHERE app_id = $1`
	args := []interface{}{appID}
	argIdx = 2

	if emailSearch != "" {
		escapedSearch := strings.ReplaceAll(emailSearch, `\`, `\\`)
		escapedSearch = strings.ReplaceAll(escapedSearch, `%`, `\%`)
		escapedSearch = strings.ReplaceAll(escapedSearch, `_`, `\_`)
		query += ` AND email ILIKE $` + itoa(argIdx)
		args = append(args, "%"+escapedSearch+"%")
		argIdx++
	}

	if providerFilter != "" {
		query += ` AND EXISTS (SELECT 1 FROM user_identities WHERE user_id = users.id AND provider = $` + itoa(argIdx) + `)`
		args = append(args, providerFilter)
		argIdx++
	}

	query += ` ORDER BY created_at DESC LIMIT $` + itoa(argIdx) + ` OFFSET $` + itoa(argIdx+1)
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(&user.ID, &user.AppID, &user.Email, &user.Name, &user.AvatarURL,
			&user.EmailVerified, &user.CreatedAt, &user.LastLoginAt)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}

	return users, totalCount, nil
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func (r *UserRepository) StoreEmailVerificationToken(ctx context.Context, userID, appID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO email_verification_tokens (user_id, app_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, userID, appID, tokenHash, expiresAt)
	return err
}

func (r *UserRepository) GetEmailVerificationToken(ctx context.Context, tokenHash, appID string) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx, `
		SELECT user_id FROM email_verification_tokens 
		WHERE token_hash = $1 AND app_id = $2 AND expires_at > NOW()
	`, tokenHash, appID).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return userID, nil
}

func (r *UserRepository) DeleteEmailVerificationToken(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM email_verification_tokens WHERE token_hash = $1
	`, tokenHash)
	return err
}
