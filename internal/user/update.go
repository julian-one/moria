package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func UpdateUsername(ctx context.Context, db *sqlx.DB, userID, username string) (*User, error) {
	var u User
	err := db.GetContext(ctx, &u,
		`UPDATE users SET username = $1, updated_at = $2
		 WHERE user_id = $3 RETURNING *`,
		username, time.Now().UTC(), userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if terr := taken(err); terr != nil {
		return nil, terr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update username: %w", err)
	}
	return &u, nil
}

func UpdateRole(ctx context.Context, db *sqlx.DB, userID string, role Role) error {
	res, err := db.ExecContext(ctx,
		`UPDATE users SET role = $1, updated_at = $2 WHERE user_id = $3`,
		role, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func UpdatePassword(ctx context.Context, db *sqlx.DB, userID, password string) error {
	h, salt, err := hash(password, nil)
	if err != nil {
		return err
	}

	res, err := db.ExecContext(ctx,
		`UPDATE users SET password_hash = $1, salt = $2, updated_at = $3
		 WHERE user_id = $4`,
		h, salt, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
