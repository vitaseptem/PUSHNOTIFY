package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword returns a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateAPIKey creates a new API key, returning the plaintext key (shown
// once), its SHA-256 hash (stored), and a short non-secret prefix.
func GenerateAPIKey() (plain, hash, prefix string, err error) {
	raw := make([]byte, 24)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("generate api key: %w", err)
	}
	body := base64.RawURLEncoding.EncodeToString(raw)
	plain = "pk_" + body
	hash = HashToken(plain)
	prefix = plain[:11]
	return plain, hash, prefix, nil
}

// HashToken returns a hex-encoded SHA-256 of a token, used for API key lookup.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RandomSecret returns a URL-safe random secret of the given byte length.
func RandomSecret(n int) (string, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("random secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
