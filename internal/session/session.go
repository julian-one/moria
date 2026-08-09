package session

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const ContextKey contextKey = "session"

const CookieName = "TOKEN"

const Duration = 24 * time.Hour

type Session struct {
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// expires_at is computed in SQL so it stores in the format datetime('now')
// comparisons expect — a bound time.Time stores RFC3339, which breaks the
// lexical comparison in ByID.
func Create(ctx context.Context, db *sql.DB, userID string) (*Session, error) {
	var s Session
	err := db.QueryRowContext(ctx,
		`INSERT INTO sessions (session_id, user_id, expires_at)
		 VALUES (?, ?, datetime('now', ?))
		 RETURNING session_id, user_id, expires_at, created_at`,
		uuid.New().String(),
		userID,
		fmt.Sprintf("+%d seconds", int(Duration/time.Second)),
	).Scan(&s.SessionID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	return &s, nil
}

func ByID(ctx context.Context, db *sql.DB, sessionID string) (*Session, error) {
	var s Session
	err := db.QueryRowContext(ctx,
		`SELECT session_id, user_id, expires_at, created_at
		 FROM sessions WHERE session_id = ? AND expires_at > datetime('now')`,
		sessionID,
	).Scan(&s.SessionID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &s, nil
}

func List(ctx context.Context, db *sql.DB, userID string) ([]Session, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT session_id, user_id, expires_at, created_at
		 FROM sessions WHERE user_id = ? ORDER BY expires_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	sessions := []Session{}
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.SessionID, &s.UserID, &s.ExpiresAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	return sessions, nil
}

func Delete(ctx context.Context, db *sql.DB, sessionID string) error {
	if _, err := db.ExecContext(ctx,
		`DELETE FROM sessions WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func DeleteAll(ctx context.Context, db *sql.DB, userID string) error {
	if _, err := db.ExecContext(ctx,
		`DELETE FROM sessions WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}
