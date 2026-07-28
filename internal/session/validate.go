package session

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Validation is the identity returned to internal callers (citadel) that
// validate an opaque session id: the session plus the owning user's role.
type Validation struct {
	SessionID string    `json:"session_id" db:"session_id"`
	UserID    string    `json:"user_id"    db:"user_id"`
	Role      string    `json:"role"       db:"role"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
}

// Validate resolves a session id to its user identity, rejecting expired sessions.
func Validate(ctx context.Context, db sqlx.QueryerContext, sessionID string) (*Validation, error) {
	var v Validation
	err := sqlx.GetContext(ctx, db, &v,
		`SELECT s.session_id, s.user_id, u.role, s.expires_at
		 FROM sessions s
		 JOIN users u ON u.user_id = s.user_id
		 WHERE s.session_id = ? AND s.expires_at > datetime('now')`,
		sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate session: %w", err)
	}
	return &v, nil
}
