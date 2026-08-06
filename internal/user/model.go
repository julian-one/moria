package user

import (
	"errors"
	"fmt"

	"moria/internal/database"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrUsernameTaken = errors.New("username taken")
	ErrEmailTaken    = errors.New("email taken")
)

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func ParseRole(s string) (Role, error) {
	switch Role(s) {
	case RoleAdmin, RoleUser:
		return Role(s), nil
	}
	return "", fmt.Errorf("invalid role %q", s)
}

type User struct {
	UserID       string           `db:"user_id"       json:"user_id"`
	Username     string           `db:"username"      json:"username"`
	Email        string           `db:"email"         json:"email"`
	PasswordHash string           `db:"password_hash" json:"-"`
	Salt         []byte           `db:"salt"          json:"-"`
	Role         Role             `db:"role"          json:"role"`
	CreatedAt    database.UTCTime `db:"created_at"    json:"created_at"`
	UpdatedAt    database.UTCTime `db:"updated_at"    json:"updated_at"`
}
