// Package security provides password hashing, encryption of secrets at rest
// (FBR security tokens), random token generation and hash chaining used for
// tamper evidence of the audit log and accepted invoices.
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password with bcrypt.
func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	return string(b), err
}

// CheckPassword compares a bcrypt hash with a password.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// PasswordPolicy enforces a minimum strength: 8+ characters with at least
// one letter and one digit.
func PasswordPolicy(pw string) error {
	if len(pw) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	var letter, digit bool
	for _, r := range pw {
		switch {
		case unicode.IsLetter(r):
			letter = true
		case unicode.IsDigit(r):
			digit = true
		}
	}
	if !letter || !digit {
		return errors.New("password must contain letters and digits")
	}
	return nil
}

// RandomToken returns a URL-safe random token of n bytes of entropy.
func RandomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// SHA256Hex returns the hex SHA-256 of s.
func SHA256Hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// ChainHash links a record to its predecessor: SHA256(prev || "|" || payload).
func ChainHash(prev, payload string) string {
	return SHA256Hex(prev + "|" + payload)
}

// ConstantTimeEqual compares two strings in constant time.
func ConstantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// Vault encrypts secrets with AES-256-GCM using a master key stored in the
// data directory (master.key, created on first run with 0600 permissions).
// Database backups therefore do not expose FBR tokens on their own.
type Vault struct {
	aead cipher.AEAD
}

// OpenVault loads or creates the master key.
func OpenVault(dataDir string) (*Vault, error) {
	path := filepath.Join(dataDir, "master.key")
	key, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		enc := []byte(hex.EncodeToString(key))
		if err := os.WriteFile(path, enc, 0o600); err != nil {
			return nil, fmt.Errorf("write master key: %w", err)
		}
		key = enc
	} else if err != nil {
		return nil, fmt.Errorf("read master key: %w", err)
	}
	raw, err := hex.DecodeString(strings.TrimSpace(string(key)))
	if err != nil || len(raw) != 32 {
		return nil, errors.New("master.key is corrupt (expected 64 hex characters)")
	}
	return NewVault(raw)
}

// NewVault builds a vault from a 32-byte key.
func NewVault(key []byte) (*Vault, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Vault{aead: aead}, nil
}

// Encrypt returns base64(nonce || ciphertext). Empty input stays empty.
func (v *Vault) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := v.aead.Seal(nonce, nonce, []byte(plain), []byte("einvoice-secret-v1"))
	return "v1:" + base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt reverses Encrypt.
func (v *Vault) Decrypt(enc string) (string, error) {
	if enc == "" {
		return "", nil
	}
	if !strings.HasPrefix(enc, "v1:") {
		return "", errors.New("unknown secret format")
	}
	raw, err := base64.StdEncoding.DecodeString(enc[3:])
	if err != nil {
		return "", err
	}
	ns := v.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("secret too short")
	}
	pt, err := v.aead.Open(nil, raw[:ns], raw[ns:], []byte("einvoice-secret-v1"))
	if err != nil {
		return "", errors.New("cannot decrypt secret (wrong master.key?)")
	}
	return string(pt), nil
}

// Mask shows only the last 4 characters of a secret.
func Mask(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 4 {
		return "****"
	}
	return strings.Repeat("•", 8) + s[len(s)-4:]
}
