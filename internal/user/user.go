package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Covers both an unknown identifier and a wrong password so callers cannot
// tell them apart.
var ErrInvalidCredentials = errors.New("invalid credentials")

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleUser:
		return true
	default:
		return false
	}
}

type User struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      Role      `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Deliberately excludes password_hash and salt so they can never leave the
// package through a handler.
const columns = `user_id, username, email, role, created_at, updated_at`

func ByID(ctx context.Context, db *sql.DB, userID string) (*User, error) {
	var u User
	err := db.QueryRowContext(ctx,
		`SELECT `+columns+` FROM users WHERE user_id = ?`,
		userID,
	).Scan(&u.UserID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func Authenticate(ctx context.Context, db *sql.DB, identifier, password string) (*User, error) {
	var (
		u    User
		hash string
		salt []byte
	)
	err := db.QueryRowContext(ctx,
		`SELECT `+columns+`, password_hash, salt FROM users
		 WHERE email = ? OR username = ?`,
		identifier, identifier,
	).Scan(&u.UserID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt, &hash, &salt)
	if errors.Is(err, sql.ErrNoRows) {
		// Burn a comparison so response timing does not reveal whether the
		// account exists.
		_, _ = verify(password, "", nil)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	match, err := verify(password, hash, salt)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, ErrInvalidCredentials
	}
	return &u, nil
}

func List(ctx context.Context, db *sql.DB) ([]User, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+columns+` FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(
			&u.UserID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

func Create(
	ctx context.Context,
	db *sql.DB,
	username, email, password string,
	role Role,
) (*User, error) {
	hash, salt, err := Hash(password, nil)
	if err != nil {
		return nil, err
	}

	var u User
	err = db.QueryRowContext(ctx,
		`INSERT INTO users (user_id, username, email, password_hash, salt, role)
		 VALUES (?, ?, ?, ?, ?, ?) RETURNING `+columns,
		uuid.New().String(), username, email, hash, salt, role,
	).Scan(&u.UserID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &u, nil
}

func IsUsernameTaken(
	ctx context.Context,
	db *sql.DB,
	username, excludeUserID string,
) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE username = ? AND user_id != ?)`,
		username, excludeUserID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check username: %w", err)
	}
	return exists, nil
}

func IsEmailTaken(
	ctx context.Context,
	db *sql.DB,
	email, excludeUserID string,
) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = ? AND user_id != ?)`,
		email, excludeUserID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email: %w", err)
	}
	return exists, nil
}

func UpdateUsername(ctx context.Context, db *sql.DB, userID, username string) (*User, error) {
	var u User
	err := db.QueryRowContext(ctx,
		`UPDATE users SET username = ?, updated_at = datetime('now')
		 WHERE user_id = ? RETURNING `+columns,
		username, userID,
	).Scan(&u.UserID, &u.Username, &u.Email, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to update username: %w", err)
	}
	return &u, nil
}

func UpdateRole(ctx context.Context, db *sql.DB, userID string, role Role) error {
	res, err := db.ExecContext(ctx,
		`UPDATE users SET role = ?, updated_at = datetime('now') WHERE user_id = ?`,
		role, userID)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func UpdatePassword(ctx context.Context, db *sql.DB, userID, password string) error {
	hash, salt, err := Hash(password, nil)
	if err != nil {
		return err
	}

	res, err := db.ExecContext(ctx,
		`UPDATE users SET password_hash = ?, salt = ?, updated_at = datetime('now')
		 WHERE user_id = ?`,
		hash, salt, userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
