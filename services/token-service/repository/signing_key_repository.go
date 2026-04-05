package repository

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"time"

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

func (r *SigningKeyRepository) GetActiveByAppID(ctx context.Context, appID string) (*SigningKey, error) {
	key := &SigningKey{}
	err := r.db.QueryRow(ctx, `
		SELECT id, app_id, kid, public_key, private_key_encrypted, is_active, created_at, expires_at, rotated_at
		FROM signing_keys
		WHERE app_id = $1 AND is_active = true
	`, appID).Scan(
		&key.ID, &key.AppID, &key.KID, &key.PublicKey, &key.PrivateKeyEncrypted,
		&key.IsActive, &key.CreatedAt, &key.ExpiresAt, &key.RotatedAt,
	)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (r *SigningKeyRepository) GetDecryptedPrivateKey(ctx context.Context, appID string) (string, string, string, error) {
	key, err := r.GetActiveByAppID(ctx, appID)
	if err != nil {
		return "", "", "", err
	}

	privateKey, err := decryptAES(key.PrivateKeyEncrypted, r.encryptionKey)
	if err != nil {
		return "", "", "", err
	}

	return privateKey, key.KID, key.AppID, nil
}

func (r *SigningKeyRepository) ListPublicKeys(ctx context.Context, appID string) ([]*SigningKey, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, app_id, kid, public_key, is_active, created_at, expires_at, rotated_at
		FROM signing_keys
		WHERE app_id = $1 AND (is_active = true OR (expires_at IS NOT NULL AND expires_at > NOW()))
		ORDER BY created_at DESC
	`, appID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*SigningKey
	for rows.Next() {
		key := &SigningKey{}
		err := rows.Scan(
			&key.ID, &key.AppID, &key.KID, &key.PublicKey,
			&key.IsActive, &key.CreatedAt, &key.ExpiresAt, &key.RotatedAt,
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// decryptAES decrypts AES-256-GCM encrypted data
func decryptAES(ciphertextB64, key string) (string, error) {
	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return "", errors.New("encryption key must be 32 bytes")
	}

	data, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("invalid ciphertext")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
