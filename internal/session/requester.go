package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"moria/internal/user"
)

type Requester struct {
	User    user.User
	Session Session
}

func RequesterByID(ctx context.Context, db *sql.DB, id ID) (*Requester, error) {
	var rq Requester
	err := db.QueryRowContext(ctx,
		`SELECT u.*, s.*
		 FROM sessions s JOIN users u ON u.user_id = s.user_id
		 WHERE s.session_id = $1 AND s.expires_at > $2`,
		id, time.Now().UTC(),
	).Scan(
		&rq.User.UserID, &rq.User.Username, &rq.User.Email, &rq.User.PasswordHash,
		&rq.User.Role, &rq.User.CreatedAt, &rq.User.UpdatedAt,
		&rq.Session.SessionID, &rq.Session.UserID,
		&rq.Session.ExpiresAt, &rq.Session.CreatedAt,
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
