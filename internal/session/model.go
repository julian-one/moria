package session

import "time"

type contextKey string

const ContextKey contextKey = "session"

type Session struct {
	SessionID string    `json:"session_id" db:"session_id"`
	User      string    `json:"user_id"    db:"user_id"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
