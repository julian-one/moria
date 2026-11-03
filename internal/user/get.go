package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func ByID(ctx context.Context, db *pgxpool.Pool, id string) (*User, error) {
	rows, err := db.Query(ctx, `
		SELECT * 
		FROM users 
		WHERE user_id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	u, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[User])
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}
