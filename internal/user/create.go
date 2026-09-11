package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

func Create(
	ctx context.Context,
	db *sql.DB,
	username, email, password string,
	role Role,
) (*User, error) {
	hash, err := hash(password)
	if err != nil {
		return nil, err
	}

	var u User
	err = u.scan(db.QueryRowContext(ctx,
		`INSERT INTO users (user_id, username, email, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5) RETURNING *`,
		uuid.New().String(), username, email, hash, role,
	))
	if terr := taken(err); terr != nil {
		return nil, terr
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return &u, nil
}
