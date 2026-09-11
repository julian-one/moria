package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"moria/schema"

	_ "github.com/lib/pq"
)

func New(ctx context.Context, url string) (*sql.DB, error) {
	db, err := sql.Open("postgres", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open the database: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("failed to ping the database: %w", err)
	}

	if _, err := db.ExecContext(ctx, schema.Model); err != nil {
		return nil, fmt.Errorf("failed to apply the schema: %w", err)
	}
	return db, nil
}
