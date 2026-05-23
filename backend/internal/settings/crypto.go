// Package settings stores app-wide configuration in the database. The
// store transparently encrypts secret string fields (anything wrapped
// in `{"__enc": true, "ciphertext": "<base64>"}`) using AES-GCM keyed
// off APP_SETTINGS_KEY. Plain-text fallback is supported when no key is
// configured, but a warning is emitted and the admin UI surfaces it.
package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// Cipher provides envelope encryption for individual JSON fields.
//
// The wire format is base64-url(nonce || ciphertext || tag) — the standard
// AES-GCM Seal output prefixed with a fresh 12-byte nonce.
type Cipher struct {
	key []byte // 32 bytes
}

// ErrNoKey is returned by NewCipher when the configured key is too short.
var ErrNoKey = errors.New("settings: APP_SETTINGS_KEY must be at least 16 characters")

// NewCipher derives a 32-byte key by SHA-256-ing the input. We accept any
// length input >= 16 chars to stay forgiving of operator habits, while
// guaranteeing a strong key regardless of entropy.
func NewCipher(secret string) (*Cipher, error) {
	if len(secret) < 16 {
		return nil, ErrNoKey
	}
	sum := sha256.Sum256([]byte(secret))
	return &Cipher{key: sum[:]}, nil
}

// Encrypt seals plaintext into a base64-url string suitable for embedding
// in JSON.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if c == nil {
		return "", errors.New("settings: cipher not configured")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. Returns an error if the ciphertext is
// truncated or tampered with.
func (c *Cipher) Decrypt(blob string) (string, error) {
	if c == nil {
		return "", errors.New("settings: cipher not configured")
	}
	raw, err := base64.RawURLEncoding.DecodeString(blob)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("settings: ciphertext too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
