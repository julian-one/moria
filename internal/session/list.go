package session

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

func List(ctx context.Context, db sqlx.QueryerContext, userID string) ([]Session, error) {
	s := []Session{}

	query, args, err := sq.Select("*").
		From("sessions").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("expires_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	err = sqlx.SelectContext(ctx, db, &s, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list session: %w", err)
	}

	return s, nil
}
