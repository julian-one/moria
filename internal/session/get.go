package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

func ByID(ctx context.Context, db *sql.DB, id ID) (*Session, error) {
	var s Session
	err := s.scan(db.QueryRowContext(ctx,
		`SELECT * FROM sessions
		 WHERE session_id = $1 AND expires_at > $2`,
		id, time.Now().UTC(),
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &s, nil
}
