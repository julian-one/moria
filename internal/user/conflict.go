package user

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

func taken(err error) error {
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23505" {
		return nil
	}
	switch pe.ConstraintName {
	case "users_username_key":
		return ErrUsernameTaken
	case "users_email_key":
		return ErrEmailTaken
	}
	return nil
}
