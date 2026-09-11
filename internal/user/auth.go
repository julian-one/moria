package user

import (
	"context"
	"database/sql"
	"errors"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

func Authenticate(ctx context.Context, db *sql.DB, identifier, password string) (*User, error) {
	u, err := ByIdentifier(ctx, db, identifier)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if !u.verifyPassword(password) {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}
