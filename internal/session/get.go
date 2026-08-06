package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func ByID(ctx context.Context, db *sqlx.DB, id ID) (*Session, error) {
	var s Session
	err := db.GetContext(ctx, &s,
		`SELECT * FROM sessions
		 WHERE session_id = $1 AND expires_at > $2`,
		id, time.Now().UTC(),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &s, nil
}
