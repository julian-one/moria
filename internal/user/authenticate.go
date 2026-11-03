package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")

	ErrInvalidIdentifier  = errors.New("invalid identifier")
	ErrIdentifierRequired = fmt.Errorf("%w: must not be empty", ErrInvalidIdentifier)
)

type LoginRequest struct {
	Identifier string
	Password   string
}

func (lr LoginRequest) Validate() error {
	if lr.Identifier == "" {
		return ErrIdentifierRequired
	}
	if lr.Password == "" {
		return ErrPasswordRequired
	}
	return nil
}

func Authenticate(ctx context.Context, db *pgxpool.Pool, request LoginRequest) (*User, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}

	rows, err := db.Query(ctx,
		`SELECT * FROM users WHERE lower(username) = lower($1) OR email = lower($1)`,
		request.Identifier,
	)
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	u, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[User])
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}

	ok, err := verify(request.Password, u.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}
