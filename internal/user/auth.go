package user

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

func Authenticate(ctx context.Context, db *sqlx.DB, identifier, password string) (*User, error) {
	u, err := ByIdentifier(ctx, db, identifier)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	match, err := u.verifyPassword(password)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}
