package repository

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/ayushdevan01/AuthService/services/developer-service/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SigningKey struct {
	ID                  string
	AppID               string
	KID                 string
	PublicKey           string
	PrivateKeyEncrypted string
	IsActive            bool
	CreatedAt           time.Time
	ExpiresAt           *time.Time
	RotatedAt           *time.Time
}

type SigningKeyRepository struct {
	db            *pgxpool.Pool
	encryptionKey string
}

func NewSigningKeyRepository(db *pgxpool.Pool, encryptionKey string) *SigningKeyRepository {
	return &SigningKeyRepository{db: db, encryptionKey: encryptionKey}
}

func generateRSAKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", "", err
	}

	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return string(publicKeyPEM), string(privateKeyPEM), nil
}

func generateKID() string {
	return fmt.Sprintf("key-%s", time.Now().Format("2006-01-02"))
}

func (r *SigningKeyRepository) Create(ctx context.Context, appID string) (*SigningKey, error) {
	publicKey, privateKey, err := generateRSAKeyPair()
	if err != nil {
		return nil, err
	}

	privateKeyEncrypted, err := auth.Encrypt(privateKey, r.encryptionKey)
	if err != nil {
		return nil, err
	}

	kid := generateKID()
	key := &SigningKey{}

	err = r.db.QueryRow(ctx, `
		INSERT INTO signing_keys (app_id, kid, public_key, private_key_encrypted, is_active)
		VALUES ($1, $2, $3, $4, true)
		RETURNING id, app_id, kid, public_key, private_key_encrypted, is_active, created_at, expires_at, rotated_at
	`, appID, kid, publicKey, privateKeyEncrypted).Scan(
		&key.ID, &key.AppID, &key.KID, &key.PublicKey, &key.PrivateKeyEncrypted, &key.IsActive, &key.CreatedAt, &key.ExpiresAt, &key.RotatedAt,
	)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (r *SigningKeyRepository) GetActiveByAppID(ctx context.Context, appID string) (*SigningKey, error) {
	key := &SigningKey{}
	err := r.db.QueryRow(ctx, `
		SELECT id, app_id, kid, public_key, private_key_encrypted, is_active, created_at, expires_at, rotated_at
		FROM signing_keys WHERE app_id = $1 AND is_active = true
	`, appID).Scan(
		&key.ID, &key.AppID, &key.KID, &key.PublicKey, &key.PrivateKeyEncrypted, &key.IsActive, &key.CreatedAt, &key.ExpiresAt, &key.RotatedAt,
	)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (r *SigningKeyRepository) GetDecryptedPrivateKey(ctx context.Context, appID string) (string, error) {
	key, err := r.GetActiveByAppID(ctx, appID)
	if err != nil {
		return "", err
	}
	return auth.Decrypt(key.PrivateKeyEncrypted, r.encryptionKey)
}

func (r *SigningKeyRepository) ListByAppID(ctx context.Context, appID string, includeExpired bool) ([]*SigningKey, error) {
	query := `
		SELECT id, app_id, kid, public_key, private_key_encrypted, is_active, created_at, expires_at, rotated_at
		FROM signing_keys WHERE app_id = $1
	`
	if !includeExpired {
		query += ` AND expires_at > NOW()`
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*SigningKey
	for rows.Next() {
		key := &SigningKey{}
		err := rows.Scan(&key.ID, &key.AppID, &key.KID, &key.PublicKey, &key.PrivateKeyEncrypted, &key.IsActive, &key.CreatedAt, &key.ExpiresAt, &key.RotatedAt)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (r *SigningKeyRepository) Rotate(ctx context.Context, appID string, gracePeriodHours int) (*SigningKey, *SigningKey, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(gracePeriodHours) * time.Hour)

	_, err := r.db.Exec(ctx, `
		UPDATE signing_keys SET is_active = false, rotated_at = $2, expires_at = $3
		WHERE app_id = $1 AND is_active = true
	`, appID, now, expiresAt)
	if err != nil {
		return nil, nil, err
	}

	oldKey := &SigningKey{}
	r.db.QueryRow(ctx, `
		SELECT id, app_id, kid, public_key, private_key_encrypted, is_active, created_at, expires_at, rotated_at
		FROM signing_keys WHERE app_id = $1 AND rotated_at = $2
	`, appID, now).Scan(
		&oldKey.ID, &oldKey.AppID, &oldKey.KID, &oldKey.PublicKey, &oldKey.PrivateKeyEncrypted, &oldKey.IsActive, &oldKey.CreatedAt, &oldKey.ExpiresAt, &oldKey.RotatedAt,
	)

	newKey, err := r.Create(ctx, appID)
	if err != nil {
		return nil, nil, err
	}

	return newKey, oldKey, nil
}
