package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"moria/internal/user"
)

type Requester struct {
	User    user.User `db:"user"`
	Session Session   `db:"session"`
}

func RequesterByID(ctx context.Context, db *sqlx.DB, id ID) (*Requester, error) {
	var rq Requester
	err := db.GetContext(ctx, &rq,
		`SELECT u.user_id    AS "user.user_id",
		        u.username   AS "user.username",
		        u.email      AS "user.email",
		        u.role       AS "user.role",
		        u.created_at AS "user.created_at",
		        u.updated_at AS "user.updated_at",
		        s.session_id AS "session.session_id",
		        s.user_id    AS "session.user_id",
		        s.expires_at AS "session.expires_at",
		        s.created_at AS "session.created_at"
		 FROM sessions s JOIN users u ON u.user_id = s.user_id
		 WHERE s.session_id = $1 AND s.expires_at > $2`,
		id, time.Now().UTC(),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get requester: %w", err)
	}
	return &rq, nil
}

func WithRequester(ctx context.Context, rq *Requester) context.Context {
	return context.WithValue(ctx, requesterKey, rq)
}

func RequesterFrom(ctx context.Context) *Requester {
	rq, ok := ctx.Value(requesterKey).(*Requester)
	if !ok {
		panic("session: requester missing from context")
	}
	return rq
}
