package database

import (
	"context"
	"fmt"
	"time"

	"moria/schema"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func New(ctx context.Context, url string) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", url)
	if err != nil {
		return nil, fmt.Errorf("failed to open the database: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("failed to ping the database: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema.Model); err != nil {
		return nil, fmt.Errorf("failed to apply the schema: %w", err)
	}
	return db, nil
}
