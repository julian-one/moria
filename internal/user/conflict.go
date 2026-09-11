package user

import (
	"errors"

	"github.com/lib/pq"
)

func taken(err error) error {
	var pe *pq.Error
	if !errors.As(err, &pe) || pe.Code != "23505" {
		return nil
	}
	switch pe.Constraint {
	case "users_username_key":
		return ErrUsernameTaken
	case "users_email_key":
		return ErrEmailTaken
	}
	return nil
}
