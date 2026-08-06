package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Purpose string

const (
	PurposeVerify Purpose = "verify"
	PurposeReset  Purpose = "reset"
)

const (
	VerifyDuration = 24 * time.Hour
	ResetDuration  = 30 * time.Minute
)

type Claims struct {
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	Purpose   Purpose `json:"purpose"`
	Binding   string  `json:"binding,omitempty"`
	ExpiresAt int64   `json:"expires_at"`
}

// CreateVerification generates a signed email-verification token.
func CreateVerification(signingKey, username, email string) (string, error) {
	return sign(signingKey, Claims{
		Username:  username,
		Email:     email,
		Purpose:   PurposeVerify,
		ExpiresAt: time.Now().Add(VerifyDuration).Unix(),
	})
}

// CreateReset generates a signed password-reset token bound to the current
// password hash, so it stops verifying once the password changes.
func CreateReset(signingKey, username, email, passwordHash string) (string, error) {
	return sign(signingKey, Claims{
		Username:  username,
		Email:     email,
		Purpose:   PurposeReset,
		Binding:   bindHash(passwordHash),
		ExpiresAt: time.Now().Add(ResetDuration).Unix(),
	})
}

// Format: base64url(JSON(claims)).base64url(HMAC-SHA256(payload, key))
func sign(signingKey string, claims Claims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)

	mac := hmac.New(sha256.New, []byte(signingKey))
	mac.Write([]byte(encodedPayload))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return encodedPayload + "." + signature, nil
}

// Verify decodes and validates a signed token.
// Returns the claims if the signature is valid, the purpose matches, and the
// token has not expired.
func Verify(signingKey, tokenString string, purpose Purpose) (*Claims, error) {
	encodedPayload, encodedSignature, ok := strings.Cut(tokenString, ".")
	if !ok {
		return nil, fmt.Errorf("invalid token format")
	}

	mac := hmac.New(sha256.New, []byte(signingKey))
	mac.Write([]byte(encodedPayload))
	expectedSig := mac.Sum(nil)

	actualSig, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	if !hmac.Equal(expectedSig, actualSig) {
		return nil, fmt.Errorf("invalid token signature")
	}

	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid token payload: %w", err)
	}

	if claims.Purpose != purpose {
		return nil, fmt.Errorf("token purpose mismatch")
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token has expired")
	}

	return &claims, nil
}

// BoundTo reports whether the token was issued against the given password hash.
func (c *Claims) BoundTo(passwordHash string) bool {
	return c.Binding != "" &&
		hmac.Equal([]byte(c.Binding), []byte(bindHash(passwordHash)))
}

// The payload is readable by the token holder, so embed a digest of the hash
// rather than the hash itself.
func bindHash(passwordHash string) string {
	sum := sha256.Sum256([]byte(passwordHash))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
