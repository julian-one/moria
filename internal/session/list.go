package session

import (
	"context"
	"database/sql"
	"fmt"
)

func List(ctx context.Context, db *sql.DB, userID string) ([]Session, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT * FROM sessions WHERE user_id = $1 ORDER BY expires_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	sessions := []Session{}
	for rows.Next() {
		var s Session
		if err := s.scan(rows); err != nil {
			return nil, fmt.Errorf("failed to list sessions: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	return sessions, nil
}
