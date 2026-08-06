package user

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/scrypt"
)

func hash(password string, salt []byte) (string, []byte, error) {
	if salt == nil {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return "", nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	h, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(h), salt, nil
}

func verify(password, storedHash string, salt []byte) (bool, error) {
	computed, _, err := hash(password, salt)
	if err != nil {
		return false, fmt.Errorf("failed to compute hash: %w", err)
	}
	return subtle.ConstantTimeCompare([]byte(computed), []byte(storedHash)) == 1, nil
}

// A nil user still burns a full scrypt derivation so response timing
// cannot reveal whether the account exists.
func (u *User) verifyPassword(password string) (bool, error) {
	if u == nil {
		_, err := verify(password, "", nil)
		return false, err
	}
	return verify(password, u.PasswordHash, u.Salt)
}
