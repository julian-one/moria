package user

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/scrypt"
)

// Hash uses scrypt with OWASP 2024 recommended parameters.
// N=32768 (2^15), r=8, p=1, keyLen=32
func Hash(password string, salt []byte) (string, []byte, error) {
	if salt == nil {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return "", nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	hash, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(hash), salt, nil
}

// Verify checks if the provided password matches the stored hash using the provided salt.
func Verify(password, storedHash string, salt []byte) (bool, error) {
	computed, _, err := Hash(password, salt)
	if err != nil {
		return false, fmt.Errorf("failed to compute hash: %w", err)
	}
	return subtle.ConstantTimeCompare([]byte(computed), []byte(storedHash)) == 1, nil
}
