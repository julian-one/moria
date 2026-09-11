package session

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

var ErrNotFound = errors.New("session not found")

type contextKey string

const (
	requesterKey contextKey    = "requester"
	CookieName   string        = "TOKEN"
	Duration     time.Duration = 24 * time.Hour
)

type Token string

type ID string

func (t Token) ID() ID {
	sum := sha256.Sum256([]byte(t))
	return ID(hex.EncodeToString(sum[:]))
}

type Session struct {
	SessionID ID        `json:"session_id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Session) scan(row interface{ Scan(...any) error }) error {
	return row.Scan(&s.SessionID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
}
