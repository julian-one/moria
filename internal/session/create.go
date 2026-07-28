package session

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func Create(
	ctx context.Context,
	db sqlx.ExtContext,
	userID string,
) (*Session, error) {
	var s Session
	err := db.QueryRowxContext(ctx,
		`INSERT INTO sessions (session_id, user_id, expires_at) VALUES (?, ?, ?) RETURNING *;`,
		uuid.New().String(),
		userID,
		time.Now().Add(SessionDuration),
	).StructScan(&s)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return &s, nil
}
