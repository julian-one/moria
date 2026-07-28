package user

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func ByID(ctx context.Context, db sqlx.QueryerContext, userID string) (*User, error) {
	var u User
	err := sqlx.GetContext(ctx, db, &u, `SELECT * FROM users WHERE user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &u, nil
}

func ByEmailOrUsername(
	ctx context.Context,
	db sqlx.QueryerContext,
	identifier string,
) (*User, error) {
	var u User
	err := sqlx.GetContext(ctx, db,
		&u,
		`SELECT * FROM users WHERE email = ? OR username = ?`,
		identifier,
		identifier,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email or username: %w", err)
	}
	return &u, nil
}

func IsUsernameTaken(ctx context.Context, db sqlx.QueryerContext, username string) (bool, error) {
	var exists bool
	err := sqlx.GetContext(
		ctx,
		db,
		&exists,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)`,
		username,
	)
	if err != nil {
		return false, fmt.Errorf("failed to check username: %w", err)
	}
	return exists, nil
}

func IsEmailTaken(ctx context.Context, db sqlx.QueryerContext, email string) (bool, error) {
	var exists bool
	err := sqlx.GetContext(
		ctx,
		db,
		&exists,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)`,
		email,
	)
	if err != nil {
		return false, fmt.Errorf("failed to check email: %w", err)
	}
	return exists, nil
}

// ByIDs retrieves the users for the given ids; missing ids are simply absent
// from the result.
func ByIDs(ctx context.Context, db sqlx.QueryerContext, ids []string) ([]User, error) {
	users := make([]User, 0)
	if len(ids) == 0 {
		return users, nil
	}

	query, args, err := sqlx.In(`SELECT * FROM users WHERE user_id IN (?)`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to build users query: %w", err)
	}

	err = sqlx.SelectContext(ctx, db, &users, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get users by ids: %w", err)
	}
	return users, nil
}
