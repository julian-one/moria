package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CreateRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Create(ctx context.Context, db sqlx.ExecerContext, request CreateRequest) (string, error) {
	h, s, err := Hash(request.Password, nil)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	uid := uuid.New().String()
	_, err = db.ExecContext(
		ctx,
		`INSERT INTO users (user_id, username, email, password_hash, salt) 
			VALUES (?, ?, ?, ?, ?)`,
		uid,
		request.Username,
		request.Email,
		h,
		s,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create user: %w", err)
	}
	return uid, nil
}
