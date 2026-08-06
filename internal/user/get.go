package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func ByID(ctx context.Context, db *sqlx.DB, userID string) (*User, error) {
	var u User
	err := db.GetContext(ctx, &u,
		`SELECT * FROM users WHERE user_id = $1`,
		userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func ByIdentifier(ctx context.Context, db *sqlx.DB, identifier string) (*User, error) {
	var u User
	err := db.GetContext(ctx, &u,
		`SELECT * FROM users
		 WHERE email = $1 OR username = $1`,
		identifier,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}
