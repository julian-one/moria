package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

func Mint(ctx context.Context, db *sqlx.DB, userID string) (*Session, Token, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, "", fmt.Errorf("failed to generate session token: %w", err)
	}
	token := Token(hex.EncodeToString(buf))

	var s Session
	err := db.GetContext(ctx, &s,
		`INSERT INTO sessions (session_id, user_id, expires_at)
		 VALUES ($1, $2, $3)
		 RETURNING *`,
		token.ID(),
		userID,
		time.Now().UTC().Add(Duration),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create session: %w", err)
	}
	return &s, token, nil
}
