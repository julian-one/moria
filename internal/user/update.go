package user

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

func Update(
	ctx context.Context,
	db sqlx.ExtContext,
	userID string,
	username *string,
	role *string,
) (*User, error) {
	query := sq.Update("users").
		Set("updated_at", sq.Expr("datetime('now')")).
		Where(sq.Eq{"user_id": userID})

	if username != nil {
		query = query.Set("username", *username)
	}
	if role != nil {
		query = query.Set("role", *role)
	}

	sql, args, err := query.
		Suffix("RETURNING *").
		PlaceholderFormat(sq.Question).
		ToSql()
	if err != nil {
		return nil, err
	}

	var user User
	err = sqlx.GetContext(ctx, db, &user, sql, args...)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func UpdatePassword(
	ctx context.Context,
	db sqlx.ExtContext,
	userID string,
	newPassword string,
) error {
	h, s, err := Hash(newPassword, nil)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	_, err = db.ExecContext(
		ctx,
		`UPDATE users SET password_hash = ?, salt = ?, updated_at = datetime('now') WHERE user_id = ?`,
		h,
		s,
		userID,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}
