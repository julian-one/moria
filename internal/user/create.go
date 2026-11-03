package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserExists    = errors.New("user already exists")
	ErrUsernameTaken = fmt.Errorf("%w: username is taken", ErrUserExists)
	ErrEmailTaken    = fmt.Errorf("%w: email is taken", ErrUserExists)

	ErrInvalidUsername  = errors.New("invalid username")
	ErrUsernameRequired = fmt.Errorf("%w: must not be empty", ErrInvalidUsername)

	ErrInvalidEmail   = errors.New("invalid email")
	ErrEmailRequired  = fmt.Errorf("%w: must not be empty", ErrInvalidEmail)
	ErrEmailMalformed = fmt.Errorf("%w: must be a valid address", ErrInvalidEmail)
)

type CreateRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (cr CreateRequest) Validate(ctx context.Context, db *pgxpool.Pool) error {
	if cr.Username == "" {
		return ErrUsernameRequired
	}
	if cr.Email == "" {
		return ErrEmailRequired
	}
	addr, err := mail.ParseAddress(cr.Email)
	if err != nil || addr.Address != cr.Email {
		return ErrEmailMalformed
	}
	err = validatePassword(cr.Password)
	if err != nil {
		return err
	}

	var usernameTaken, emailTaken bool
	err = db.QueryRow(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM users WHERE username = $1),
			EXISTS (SELECT 1 FROM users WHERE email = $2)`,
		cr.Username, cr.Email,
	).Scan(&usernameTaken, &emailTaken)
	if err != nil {
		return fmt.Errorf("check existing user: %w", err)
	}

	if usernameTaken {
		return ErrUsernameTaken
	}
	if emailTaken {
		return ErrEmailTaken
	}
	return nil
}

func Create(
	ctx context.Context,
	db *pgxpool.Pool,
	request CreateRequest,
) (*User, error) {
	request.Email = strings.ToLower(request.Email)

	if err := request.Validate(ctx, db); err != nil {
		return nil, err
	}

	passwordHash, err := hash(request.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	rows, err := db.Query(ctx, `
		INSERT INTO users (user_id, username, email, password_hash)
		VALUES (gen_random_uuid()::text, $1, $2, $3)
		RETURNING *`, request.Username, request.Email, passwordHash)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	u, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[User])
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}
