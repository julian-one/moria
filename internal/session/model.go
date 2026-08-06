package session

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"moria/internal/database"
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
	SessionID ID               `db:"session_id" json:"session_id"`
	UserID    string           `db:"user_id"    json:"user_id"`
	ExpiresAt database.UTCTime `db:"expires_at" json:"expires_at"`
	CreatedAt database.UTCTime `db:"created_at" json:"created_at"`
}
