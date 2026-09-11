package user

import (
	"errors"
	"fmt"
	"time"
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
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) scan(row interface{ Scan(...any) error }) error {
	return row.Scan(
		&u.UserID, &u.Username, &u.Email, &u.PasswordHash,
		&u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
}
