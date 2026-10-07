package application

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type PasswordHasher struct {
	MinLength int
	MaxLength int
}

func (h PasswordHasher) Validate(password string) error {
	if len([]rune(password)) < h.MinLength || len([]rune(password)) > h.MaxLength {
		return fmt.Errorf("password must be between %d and %d characters", h.MinLength, h.MaxLength)
	}
	return nil
}

func (h PasswordHasher) Hash(password string) (string, error) {
	if err := h.Validate(password); err != nil {
		return "", err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=1,p=4$%s$%s", encode(salt), encode(key)), nil
}

func (h PasswordHasher) Verify(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=65536,t=1,p=4" {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func encode(value []byte) string { return base64.RawStdEncoding.EncodeToString(value) }
