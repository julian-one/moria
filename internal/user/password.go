package user

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/scrypt"
)

const (
	minPasswordLen = 8
	saltLen        = 32
	keyLen         = 32
	scryptN        = 16384
	scryptR        = 8
	scryptP        = 1
)

var errMalformedHash = errors.New("parse password hash: missing separator")

var (
	ErrInvalidPassword  = errors.New("invalid password")
	ErrPasswordRequired = fmt.Errorf("%w: must not be empty", ErrInvalidPassword)
	ErrPasswordTooShort = fmt.Errorf(
		"%w: must be at least %d characters",
		ErrInvalidPassword,
		minPasswordLen,
	)
	ErrPasswordNoUppercase = fmt.Errorf(
		"%w: must contain an uppercase letter",
		ErrInvalidPassword,
	)
	ErrPasswordNoLowercase   = fmt.Errorf("%w: must contain a lowercase letter", ErrInvalidPassword)
	ErrPasswordNoSpecialChar = fmt.Errorf(
		"%w: must contain a special character",
		ErrInvalidPassword,
	)
)

func validatePassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}
	if utf8.RuneCountInString(password) < minPasswordLen {
		return ErrPasswordTooShort
	}
	if !strings.ContainsFunc(password, unicode.IsUpper) {
		return ErrPasswordNoUppercase
	}
	if !strings.ContainsFunc(password, unicode.IsLower) {
		return ErrPasswordNoLowercase
	}
	if !strings.ContainsFunc(password, isSpecial) {
		return ErrPasswordNoSpecialChar
	}
	return nil
}

func isSpecial(r rune) bool {
	return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsSpace(r)
}

func hash(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return "", fmt.Errorf("derive key: %w", err)
	}
	enc := base64.StdEncoding
	return enc.EncodeToString(salt) + ":" + enc.EncodeToString(key), nil
}

func verify(password, encoded string) (bool, error) {
	encodedSalt, encodedKey, found := strings.Cut(encoded, ":")
	if !found {
		return false, errMalformedHash
	}
	enc := base64.StdEncoding
	salt, err := enc.DecodeString(encodedSalt)
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}
	want, err := enc.DecodeString(encodedKey)
	if err != nil {
		return false, fmt.Errorf("decode key: %w", err)
	}
	got, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keyLen)
	if err != nil {
		return false, fmt.Errorf("derive key: %w", err)
	}
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
