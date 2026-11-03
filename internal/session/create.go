package session

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const TTL = 30 * 24 * time.Hour

func Create(ctx context.Context, db *pgxpool.Pool, userID string) (*Session, error) {
	rows, err := db.Query(ctx, `
		INSERT INTO sessions (session_id, user_id, expires_at)
		VALUES (gen_random_uuid()::text, $1, $2)
		RETURNING *`,
		userID, time.Now().Add(TTL),
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}
	s, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[Session])
	if err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}
	return s, nil
}
