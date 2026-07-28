package session

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Delete removes a specific session (logout).
func Delete(ctx context.Context, db sqlx.ExecerContext, sessionID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM sessions WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteAll removes all sessions for a user (logout everywhere).
func DeleteAll(ctx context.Context, db sqlx.ExecerContext, userID string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}
