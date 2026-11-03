package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/julian-one/moria/internal/user"
)

var ErrInvalidSession = errors.New("invalid session: not found or expired")

func Verify(ctx context.Context, db *pgxpool.Pool, token string) (*user.User, error) {
	if token == "" {
		return nil, ErrInvalidSession
	}
	rows, err := db.Query(ctx, `
		SELECT u.*
		FROM sessions s
		JOIN users u USING (user_id)
		WHERE s.session_id = $1 AND s.expires_at > now()`, token)
	if err != nil {
		return nil, fmt.Errorf("query session user: %w", err)
	}
	u, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[user.User])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidSession
	}
	if err != nil {
		return nil, fmt.Errorf("scan session user: %w", err)
	}
	return u, nil
}
