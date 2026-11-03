package session

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Delete(ctx context.Context, db *pgxpool.Pool, token string) error {
	_, err := db.Exec(ctx, `DELETE FROM sessions WHERE session_id = $1`, token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}
