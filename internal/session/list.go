package session

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func List(ctx context.Context, db *sqlx.DB, userID string) ([]Session, error) {
	sessions := []Session{}
	err := db.SelectContext(ctx, &sessions,
		`SELECT * FROM sessions WHERE user_id = $1 ORDER BY expires_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	return sessions, nil
}
