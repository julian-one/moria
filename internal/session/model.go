package session

import "time"

type Session struct {
	SessionID string    `db:"session_id" json:"session_id"`
	UserID    string    `db:"user_id"    json:"user_id"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
