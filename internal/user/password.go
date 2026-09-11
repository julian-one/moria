package user

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const hashCost = 12

var decoyHash []byte

func init() {
	h, err := bcrypt.GenerateFromPassword([]byte("decoy"), hashCost)
	if err != nil {
		panic(err)
	}
	decoyHash = h
}

func hash(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(h), nil
}

func (u *User) verifyPassword(password string) bool {
	if u == nil {
		_ = bcrypt.CompareHashAndPassword(decoyHash, []byte(password))
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}
