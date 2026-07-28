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

const Duration = 24 * time.Hour

type Claims struct {
	Username  string `json:"username"`
	Email     string `json:"email"`
	ExpiresAt int64  `json:"expires_at"`
}

// Create generates a signed token containing the username and email.
// Format: base64url(JSON(claims)).base64url(HMAC-SHA256(payload, key))
func Create(signingKey, username, email string) (string, error) {
	claims := Claims{
		Username:  username,
		Email:     email,
		ExpiresAt: time.Now().Add(Duration).Unix(),
	}

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
// Returns the claims if the signature is valid and the token has not expired.
func Verify(signingKey, tokenString string) (*Claims, error) {
	encodedPayload, encodedSignature, ok := strings.Cut(tokenString, ".")
	if !ok {
		return nil, fmt.Errorf("invalid token format")
	}

	// Verify HMAC signature
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

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, fmt.Errorf("invalid payload encoding: %w", err)
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid token payload: %w", err)
	}

	// Check expiry
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token has expired")
	}

	return &claims, nil
}
