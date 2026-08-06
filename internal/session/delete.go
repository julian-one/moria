package session

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func Delete(ctx context.Context, db *sqlx.DB, id ID) error {
	if _, err := db.ExecContext(ctx,
		`DELETE FROM sessions WHERE session_id = $1`, id); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func DeleteAll(ctx context.Context, db *sqlx.DB, userID string) error {
	if _, err := db.ExecContext(ctx,
		`DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func DeleteExpired(ctx context.Context, db *sqlx.DB) error {
	if _, err := db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at < $1`, time.Now().UTC()); err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}
