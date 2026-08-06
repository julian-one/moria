package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func Create(
	ctx context.Context,
	db *sqlx.DB,
	username, email, password string,
	role Role,
) (*User, error) {
	hash, salt, err := hash(password, nil)
	if err != nil {
		return nil, err
	}

	var u User
	err = db.GetContext(ctx, &u,
		`INSERT INTO users (user_id, username, email, password_hash, salt, role)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING *`,
		uuid.New().String(), username, email, hash, salt, role,
	)
	if terr := taken(err); terr != nil {
		return nil, terr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &u, nil
}
