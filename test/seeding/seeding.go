package seeding

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"moria/internal/session"
	"moria/internal/user"
)

const (
	AdminID    = "seed-admin-id"
	AdminToken = "seed-admin-token"
	AliceID    = "seed-alice-id"
	AliceToken = "seed-alice-token"

	DefaultPassword = "password"
)

// One scrypt derivation per test binary (~100ms each); every seeded user
// shares it. The derivation stays inside internal/user (its hash func is
// unexported), so the first seed goes through user.Create and later seeds
// reuse the hash and salt it returned.
var creds struct {
	sync.Mutex
	hash string
	salt []byte
}

func Defaults(db *sqlx.DB) error {
	if err := User(db, AdminID, "admin", user.RoleAdmin); err != nil {
		return err
	}
	if err := User(db, AliceID, "alice", user.RoleUser); err != nil {
		return err
	}
	if err := Session(db, AdminToken, AdminID); err != nil {
		return err
	}
	return Session(db, AliceToken, AliceID)
}

func User(db *sqlx.DB, id, username string, role user.Role) error {
	creds.Lock()
	defer creds.Unlock()

	if creds.hash == "" {
		u, err := user.Create(
			context.Background(), db, username, username+"@example.com",
			DefaultPassword, role)
		if err != nil {
			return fmt.Errorf("seed user %s: %w", username, err)
		}
		_, err = db.Exec(
			`UPDATE users SET user_id = $1 WHERE user_id = $2`, id, u.UserID)
		if err != nil {
			return fmt.Errorf("seed user %s: %w", username, err)
		}
		creds.hash, creds.salt = u.PasswordHash, u.Salt
		return nil
	}

	_, err := db.Exec(
		`INSERT INTO users (user_id, username, email, password_hash, salt, role)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		id, username, username+"@example.com", creds.hash, creds.salt, role)
	if err != nil {
		return fmt.Errorf("seed user %s: %w", username, err)
	}
	return nil
}

func Session(db *sqlx.DB, token, userID string) error {
	_, err := db.Exec(
		`INSERT INTO sessions (session_id, user_id, expires_at)
		 VALUES ($1, $2, $3)`,
		session.Token(token).ID(), userID, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		return fmt.Errorf("seed session for %s: %w", userID, err)
	}
	return nil
}

func ExpiredSession(db *sqlx.DB, token, userID string) error {
	_, err := db.Exec(
		`INSERT INTO sessions (session_id, user_id, expires_at)
		 VALUES ($1, $2, $3)`,
		session.Token(token).ID(), userID, time.Now().UTC().Add(-time.Hour))
	if err != nil {
		return fmt.Errorf("seed expired session for %s: %w", userID, err)
	}
	return nil
}
